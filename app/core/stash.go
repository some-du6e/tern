package core

import (
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
	a.saveStashLocked()
}

// saveStashLocked writes the stash, dropping the pictures past what it
// keeps. Call with the lock held.
func (a *App) saveStashLocked() {
	room := stashImageBytes
	for i := range a.comp.stash {
		e := &a.comp.stash[i]
		var keep []clip.Image
		for _, im := range e.Images {
			if len(im.Data) > room {
				e.Dropped++
				continue
			}
			room -= len(im.Data)
			keep = append(keep, im)
		}
		if len(keep) < len(e.Images) {
			e.Images, e.thumbs = keep, nil
		}
	}
	_ = a.store.Put(stashFile, a.comp.stash)
}

// restoreStash brings entry i back: its text after the composer's (a
// blank line between), its pictures as tiles, and takes it off the stash.
func (a *App) restoreStash(th *ui.Theme, i int) {
	a.mu.Lock()
	if i < 0 || i >= len(a.comp.stash) {
		a.mu.Unlock()
		return
	}
	e := a.comp.stash[i]
	a.comp.stash = append(a.comp.stash[:i:i], a.comp.stash[i+1:]...)
	a.comp.stashOpen = false
	a.saveStashLocked()
	key := a.composerKey()
	for j, im := range e.Images {
		n := a.freeNumber(key)
		if n == 0 {
			break
		}
		p := pastedImage{id: t3.NewID(), n: n, img: im}
		if j < len(e.thumbs) {
			p.thumb = e.thumbs[j]
		} else {
			p.thumb = ui.NewThumb(im.Data)
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
func (a *App) stashScene(s *ui.Scene) {
	for i := range a.comp.stash {
		e := &a.comp.stash[i]
		if e.thumbs == nil {
			for _, im := range e.Images {
				e.thumbs = append(e.thumbs, ui.NewThumb(im.Data))
			}
		}
		s.Stash = append(s.Stash, ui.Stashed{Snippet: stashSnippet(*e), Thumbs: e.thumbs, Dropped: e.Dropped, Age: age(e.Created)})
	}
	s.StashOpen = a.comp.stashOpen && len(s.Suggest) == 0 && s.SuggestEmpty == ""
	s.StashSel = a.comp.stashSel
}
