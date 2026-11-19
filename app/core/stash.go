package core

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/some-du6e/tern/app/clip"
	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The composer's prompt stash (useComposerStash.ts, promptStashStore.ts):
// Mod+S puts the composer's prompt and pictures aside and clears it; with
// nothing in the composer it brings back the only entry, or opens the
// list. Entries persist, 20 at most, newest first.

const (
	stashMax = 20 // promptStashStore.ts
	// stashImageBytes is how much picture data the stash keeps on disk;
	// pictures past it are dropped, as T3's localStorage drops them.
	stashImageBytes = 8 << 20
)

// stashEntry is one stashed prompt. Its pictures come back as tiles.
type stashEntry struct {
	ID      string       `json:"id"`
	Prompt  string       `json:"prompt"`
	Images  []clip.Image `json:"images,omitempty"`
	Dropped int          `json:"dropped,omitempty"`
	Created time.Time    `json:"createdAt"`
	thumbs  []ui.Thumb
}

// stashFile is where the stash persists.
const stashFile = "prompt-stash.json"

// stashSnippet is the entry's line in the list (ComposerStashMenu.tsx):
// its text with the white space collapsed, cut at 90 characters, or what
// it carries.
func stashSnippet(e stashEntry) string {
	text := strings.Join(strings.FieldsFunc(ui.StripChips(e.Prompt), unicode.IsSpace), " ")
	if r := []rune(text); len(r) > 90 {
		return string(r[:90]) + "…"
	}
	if text != "" {
		return text
	}
	switch n := len(e.Images) + e.Dropped; n {
	case 0:
		return "(empty)"
	case 1:
		return "(1 image)"
	default:
		return "(" + strconv.Itoa(n) + " images)"
	}
}

// stashKey is Mod+S. With a question open it only opens or closes the list.
func (a *App) stashKey(th *ui.Theme) {
	text := ui.TrimText(ui.StripChips(th.EditorText("composer")))
	a.mu.Lock()
	key := a.composerKey()
	asking := len(a.questionsLocked()) > 0
	switch {
	case key == "":
	case !asking && (text != "" || len(a.images[key]) > 0):
		a.stashLocked(th, key)
		a.mu.Unlock()
		th.SetEditorText("composer", "")
		return
	case !asking && len(a.comp.stash) == 1:
		a.mu.Unlock()
		a.restoreStash(th, 0)
		return
	default:
		a.comp.stashOpen = !a.comp.stashOpen
		a.comp.stashSel = 0
	}
	a.mu.Unlock()
}

// stashLocked puts the composer's prompt and pictures at the top of the
// stash. Call with the lock held.
func (a *App) stashLocked(th *ui.Theme, key string) {
	e := stashEntry{ID: t3.NewID(), Prompt: ui.TrimText(ui.StripChips(th.EditorText("composer"))), Created: time.Now()}
	for _, p := range a.images[key] {
		e.Images = append(e.Images, p.img)
		e.thumbs = append(e.thumbs, p.thumb)
	}
	delete(a.images, key)
	delete(a.trash, key)
	a.comp.stash = append([]stashEntry{e}, a.comp.stash...)
	if len(a.comp.stash) > stashMax {
		a.comp.stash = a.comp.stash[:stashMax]
	}
	a.comp.stashOpen = false
	a.comp.stashed = time.Now()
	a.saveStashLocked()
}

// saveStashLocked drops the pictures past what the stash keeps and writes
// it off the UI's path, the newest copy last. Call with the lock held.
func (a *App) saveStashLocked() {
	room := stashImageBytes
	for i := range a.comp.stash {
		e := &a.comp.stash[i]
		var keep []clip.Image
		var thumbs []ui.Thumb
		for j, im := range e.Images {
			if len(im.Data) > room {
				e.Dropped++
				continue
			}
			room -= len(im.Data)
			keep = append(keep, im)
			if j < len(e.thumbs) {
				thumbs = append(thumbs, e.thumbs[j])
			}
		}
		if len(keep) < len(e.Images) {
			e.Images, e.thumbs = keep, thumbs
		}
	}
	a.stashWrites++
	gen, snapshot := a.stashWrites, slices.Clone(a.comp.stash)
	a.stashSaving.Add(1)
	go func() {
		defer a.stashSaving.Done()
		a.stashFileMu.Lock()
		defer a.stashFileMu.Unlock()
		if gen > a.stashWritten {
			a.stashWritten = gen
			_ = a.store.Put(stashFile, snapshot)
		}
	}()
}

// decodeStash makes the stashed pictures' thumbnails, off the UI's path,
// for a stash read back from disk.
func (a *App) decodeStash() {
	a.mu.Lock()
	entries := slices.Clone(a.comp.stash)
	a.mu.Unlock()
	thumbs := map[string][]ui.Thumb{}
	for _, e := range entries {
		for _, im := range e.Images {
			t := ui.NewThumb(im.Data)
			t.Name = im.Name
			thumbs[e.ID] = append(thumbs[e.ID], t)
		}
	}
	a.mu.Lock()
	for i := range a.comp.stash {
		if e := &a.comp.stash[i]; e.thumbs == nil && len(thumbs[e.ID]) == len(e.Images) {
			e.thumbs = thumbs[e.ID]
		}
	}
	a.mu.Unlock()
	a.changed()
}

// restoreStash brings entry i back: its text after the composer's (a
// blank line between), its pictures as tiles, and takes it off the stash.
// When the composer hasn't room for all its pictures it stays stashed.
func (a *App) restoreStash(th *ui.Theme, i int) {
	a.mu.Lock()
	if i < 0 || i >= len(a.comp.stash) {
		a.mu.Unlock()
		return
	}
	e := a.comp.stash[i]
	key := a.imageKey()
	if len(a.images[key])+len(e.Images) > maxImages {
		a.fail(errors.New("a message takes " + strconv.Itoa(maxImages) + " images at most, so this one stays stashed"))
		a.mu.Unlock()
		a.changed()
		return
	}
	a.comp.stash = append(a.comp.stash[:i:i], a.comp.stash[i+1:]...)
	a.comp.stashOpen = false
	a.saveStashLocked()
	for j, im := range e.Images {
		p := pastedImage{id: t3.NewID(), n: a.freeNumber(key), img: im}
		if j < len(e.thumbs) {
			p.thumb = e.thumbs[j]
		} else {
			p.thumb = ui.NewThumb(im.Data)
			p.thumb.Name = im.Name
		}
		p.thumb.N = 0
		a.images[key] = append(a.images[key], p)
	}
	a.mu.Unlock()
	text := e.Prompt
	if cur := strings.TrimRightFunc(th.EditorText("composer"), unicode.IsSpace); cur != "" {
		text = cur + "\n\n" + text
	}
	th.SetEditorText("composer", text)
	a.changed()
}

// deleteStash takes entry i off the stash.
func (a *App) deleteStash(i int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if i < 0 || i >= len(a.comp.stash) {
		return
	}
	a.comp.stash = append(a.comp.stash[:i:i], a.comp.stash[i+1:]...)
	a.comp.stashSel = min(a.comp.stashSel, max(0, len(a.comp.stash)-1))
	a.saveStashLocked()
}

// stashMenuKey is the open list's keys: the arrows move the highlight
// around, Enter restores it, Mod+Backspace deletes it, Escape closes.
func (a *App) stashMenuKey(th *ui.Theme, name string) {
	a.mu.Lock()
	n := len(a.comp.stash)
	sel := a.comp.stashSel
	switch name {
	case "↑", "↓":
		if n > 0 {
			step := 1
			if name == "↑" {
				step = n - 1
			}
			a.comp.stashSel = (sel + step) % n
		}
	case "⎋":
		a.comp.stashOpen = false
	}
	a.mu.Unlock()
	switch name {
	case "⏎", "⌤":
		a.restoreStash(th, sel)
	case "⌫":
		a.deleteStash(sel)
	}
}

// stashScene shows the stash, its list unless a menu is open. Call with
// the lock held.
func (a *App) stashScene(s *ui.Scene, now time.Time) {
	for _, e := range a.comp.stash {
		s.Stash = append(s.Stash, ui.Stashed{Snippet: stashSnippet(e), Thumbs: e.thumbs, Dropped: e.Dropped, Age: relativeAge(e.Created, now)})
	}
	s.StashOpen = a.comp.stashOpen && len(s.Suggest) == 0 && s.SuggestEmpty == ""
	s.StashSel = a.comp.stashSel
	s.Stashed = a.comp.stashed
}

// relativeAge is T3's RelativeTime (formatRelativeElapsed): "just now",
// then "4m ago", "1h ago", "2d ago".
func relativeAge(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	switch d := max(0, now.Sub(t)); {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return strconv.Itoa(int(d/time.Minute)) + "m ago"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + "h ago"
	default:
		return strconv.Itoa(int(d/(24*time.Hour))) + "d ago"
	}
}
