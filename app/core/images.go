package core

import (
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/some-du6e/tern/app/clip"
	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// maxImages is how many pictures one message carries.
const maxImages = 8

type pastedImage struct {
	id string // the composer's own id, which names its context record
	n  int    // its number in the composer: ChipText(n) in the text
	// pending is true until the chip is typed into the composer.
	pending bool
	img     clip.Image
	thumb   ui.Thumb
}

// readClipboard is clip.Images, swapped out in tests.
var readClipboard = clip.Images

// paste is the composer's Ctrl+V: a picture on the clipboard goes on the
// message, anything else pastes as text.
func (a *App) paste(th *ui.Theme) {
	a.mu.Lock()
	key := a.composerKey()
	a.mu.Unlock()
	if key == "" {
		th.PasteText("composer")
		return
	}
	go func() {
		imgs, err := readClipboard()
		a.pasted(key, imgs, err)
	}()
}

// pasted files what the clipboard gave under the composer it came from.
func (a *App) pasted(key string, imgs []clip.Image, err error) {
	if errors.Is(err, clip.ErrNone) {
		a.onUI(func(th *ui.Theme) { th.PasteText("composer") })
		return
	}
	thumbs := make([]ui.Thumb, len(imgs))
	for i, im := range imgs {
		thumbs[i] = ui.NewThumb(im.Data)
	}
	a.mu.Lock()
	if err != nil {
		a.fail(err)
	}
	var added []string
	for i, im := range imgs {
		n := a.freeNumber(key)
		if n == 0 {
			a.fail(errors.New("a message takes " + strconv.Itoa(maxImages) + " images at most"))
			break
		}
		p := pastedImage{id: t3.NewID(), n: n, img: im, thumb: thumbs[i], pending: true}
		p.thumb.N = n
		a.images[key] = append(a.images[key], p)
		added = append(added, p.id)
	}
	a.mu.Unlock()
	a.changed()
	if len(added) == 0 {
		return
	}
	// Like the web's chips, a picture goes where the caret is and is sent at
	// that spot.
	a.onUI(func(th *ui.Theme) {
		a.mu.Lock()
		here := a.composerKey() == key
		markers := ""
		for i := range a.images[key] {
			p := &a.images[key][i]
			if p.pending && slices.Contains(added, p.id) {
				p.pending = false
				markers += p.marker()
			}
		}
		a.mu.Unlock()
		if here {
			th.InsertText("composer", markers)
		} else {
			saveDraft(key, drafts[key]+markers)
		}
	})
}

// freeNumber is the lowest picture number the composer isn't using, or 0
// when it's full. A number taken again forgets the deleted picture that had
// it, so undo can't bring that one back.
func (a *App) freeNumber(key string) int {
	if len(a.images[key]) >= maxImages {
		return 0
	}
	used := map[int]bool{}
	for _, p := range a.images[key] {
		used[p.n] = true
	}
	for n := 1; n <= 9; n++ {
		if !used[n] {
			a.trash[key] = slices.DeleteFunc(a.trash[key], func(p pastedImage) bool { return p.n == n })
			return n
		}
	}
	return 0
}

func (p pastedImage) marker() string { return ui.ChipText(p.n) }
func (p pastedImage) label() string  { return ui.ImageLabel(p.n) }

// syncImages keeps the pictures in step with the composer's text: deleting
// a picture's chip drops it, and undo bringing the chip back restores it.
func (a *App) syncImages(text string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	key := a.composerKey()
	if key == "" || len(a.images[key])+len(a.trash[key]) == 0 {
		return
	}
	var keep []pastedImage
	for _, p := range a.images[key] {
		if p.pending || strings.Contains(text, p.marker()) {
			keep = append(keep, p)
		} else {
			a.trash[key] = append(a.trash[key], p)
		}
	}
	var trash []pastedImage
	for _, p := range a.trash[key] {
		if strings.Contains(text, p.marker()) && len(keep) < maxImages {
			keep = append(keep, p)
		} else {
			trash = append(trash, p)
		}
	}
	slices.SortStableFunc(keep, func(x, y pastedImage) int { return x.n - y.n })
	a.images[key], a.trash[key] = keep, trash
	if len(keep) == 0 {
		delete(a.images, key)
	}
	if len(trash) == 0 {
		delete(a.trash, key)
	}
}

// unpaste drops the composer's image i and its chip.
func (a *App) unpaste(th *ui.Theme, arg string) {
	i, err := strconv.Atoi(arg)
	a.mu.Lock()
	key := a.composerKey()
	imgs := a.images[key]
	if err != nil || i < 0 || i >= len(imgs) {
		a.mu.Unlock()
		return
	}
	gone := imgs[i]
	a.images[key] = append(imgs[:i:i], imgs[i+1:]...)
	if len(a.images[key]) == 0 {
		delete(a.images, key)
	}
	a.mu.Unlock()
	th.DeleteChip("composer", gone.n)
}

// upload puts the images on the environment, in order, and returns the
// message's attachments and their context records.
func upload(conn *t3.Conn, imgs []pastedImage) ([]any, []t3.ImageRecord, error) {
	out := []any{}
	var recs []t3.ImageRecord
	for _, p := range imgs {
		at, err := conn.UploadImage(p.img.Name, p.img.Mime, p.img.Data)
		if err != nil {
			return nil, nil, errors.New("couldn't upload " + p.img.Name + ": " + err.Error())
		}
		out = append(out, at)
		recs = append(recs, t3.NewImageRecord(p.id, p.label(), at))
	}
	return out, recs, nil
}

// message is a prompt's text and context fields with its pictures in them,
// the way the web sends them: each picture's chip becomes its token, one
// without a chip goes at the end, and the records go
// in context. A server without inline context gets the text the agent would.
func message(body string, imgs []pastedImage, recs []t3.ImageRecord, inline bool) map[string]any {
	if len(recs) == 0 {
		return map[string]any{"text": body}
	}
	var rest []t3.ImageRecord
	for i, r := range recs {
		if m := imgs[i].marker(); strings.Contains(body, m) {
			body = strings.Replace(body, m, r.Token(), 1)
		} else {
			rest = append(rest, r)
		}
	}
	text := t3.WithImages(body, rest)
	if !inline {
		return map[string]any{"text": t3.ProjectImages(text, recs)}
	}
	return map[string]any{"text": text, "context": map[string]any{"version": 1, "records": recs}}
}

// mergeMap copies src's keys into dst and returns dst.
func mergeMap(dst, src map[string]any) map[string]any {
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// sentThumb is a sent picture, loaded once per attachment.
type sentThumb struct {
	picture ui.Picture
	loading bool
	failed  bool
}

// fetchSlots caps how many sent pictures download at once.
var fetchSlots = make(chan struct{}, 3)

// attachmentPictures is a message's pictures, starting the download of any
// not loaded yet. Call with the lock held.
func (a *App) attachmentPictures(e *envState, raws []json.RawMessage) []ui.Picture {
	var out []ui.Picture
	for _, raw := range raws {
		var at t3.Attachment
		if json.Unmarshal(raw, &at) != nil || at.Type != "image" || at.ID == "" {
			continue
		}
		key := e.id() + "/" + at.ID
		st := a.sent[key]
		if st == nil && e.conn != nil {
			st = &sentThumb{loading: true}
			a.sent[key] = st
			go a.fetchPicture(e.conn, at.ID, st)
		}
		var p ui.Picture
		if st != nil {
			p = st.picture
		}
		p.Name = at.Name
		out = append(out, p)
	}
	return out
}

func (a *App) fetchPicture(conn *t3.Conn, id string, st *sentThumb) {
	fetchSlots <- struct{}{}
	data, err := conn.FetchAttachment(id)
	<-fetchSlots
	var p ui.Picture
	if err == nil {
		p = ui.NewPicture(data)
	}
	a.mu.Lock()
	st.picture, st.loading, st.failed = p, false, err != nil
	a.mu.Unlock()
	a.changed()
}
