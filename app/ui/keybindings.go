package ui

import (
	"runtime"
	"strconv"
	"strings"

	"gioui.org/io/event"
	"gioui.org/io/key"
)

// T3's keybindings (apps/web/src/keybindings.ts, packages/shared/src/
// keybindings.ts): the server's rules after the defaults for the commands
// they don't rebind. A key goes to the last rule that holds and matches
// it, and a command's label is its last shortcut that holds and that no
// later rule has taken.

// Keybinding is one rule: its command, its key as T3 names it ("b",
// "escape", "f6"), the modifiers it wants (mod is Cmd on a Mac, Ctrl
// elsewhere), and the condition it holds in.
type Keybinding struct {
	Command                     string
	Key                         string
	Mod, Meta, Ctrl, Shift, Alt bool
	When                        *When
}

// When is a rule's condition (KeybindingWhenNode): Op "identifier" names
// a context value, "not" negates Left, "and" and "or" join Left and Right.
type When struct {
	Op, Name    string
	Left, Right *When
}

// defaultKeys are T3's defaults for the commands the shell runs
// (DEFAULT_KEYBINDINGS), in T3's order. None shares a key with another of
// T3's defaults, so merging only these gives these commands the labels
// and keys T3's whole table does.
var defaultKeys = []Keybinding{
	{Command: "sidebar.toggle", Key: "b", Mod: true},
	{Command: "terminal.toggle", Key: "j", Mod: true},
	{Command: "rightPanel.toggle", Key: "b", Mod: true, Alt: true},
	{Command: "threadPanel.toggle", Key: "b", Mod: true, Shift: true, When: &When{Op: "not", Left: &When{Op: "identifier", Name: "terminalFocus"}}},
	{Command: "editor.openFavorite", Key: "o", Mod: true},
}

// MergeKeys is the server's rules after the defaults for the commands
// they leave alone (mergeWithDefaultKeybindings): the defaults alone when
// there are none.
func MergeKeys(custom []Keybinding) []Keybinding {
	rebound := map[string]bool{}
	for _, k := range custom {
		rebound[k.Command] = true
	}
	var out []Keybinding
	for _, k := range defaultKeys {
		if !rebound[k.Command] {
			out = append(out, k)
		}
	}
	return append(out, custom...)
}

// holds reports whether w holds in ctx (evaluateWhenNode): true and false
// are themselves, any other name its value in ctx; no condition always
// holds.
func (w *When) holds(ctx map[string]bool) bool {
	if w == nil {
		return true
	}
	switch w.Op {
	case "identifier":
		if w.Name == "true" || w.Name == "false" {
			return w.Name == "true"
		}
		return ctx[w.Name]
	case "not":
		return !w.Left.holds(ctx)
	case "and":
		return w.Left.holds(ctx) && w.Right.holds(ctx)
	case "or":
		return w.Left.holds(ctx) || w.Right.holds(ctx)
	}
	return false
}

// labelContext is what a label is worked out in: tern is T3's desktop app,
// with nothing focused.
var labelContext = map[string]bool{"isDesktop": true}

// mods is the modifiers k wants on this computer: mod and meta are Cmd on
// a Mac; elsewhere mod is Ctrl and meta the Super key.
func (k Keybinding) mods() key.Modifiers {
	mac := runtime.GOOS == "darwin"
	var m key.Modifiers
	if k.Meta && mac || k.Mod && mac {
		m |= key.ModCommand
	}
	if k.Meta && !mac {
		m |= key.ModSuper
	}
	if k.Ctrl || k.Mod && !mac {
		m |= key.ModCtrl
	}
	if k.Shift {
		m |= key.ModShift
	}
	if k.Alt {
		m |= key.ModAlt
	}
	return m
}

// claim is k's key and modifiers as one value, which only the last rule
// that holds may take (shortcutConflictKey).
func (k Keybinding) claim() string { return k.Key + "|" + strconv.Itoa(int(k.mods())) }

// effective is command's shortcuts that hold in ctx and that no later rule
// takes (effectiveShortcutsForCommand), the last rule's first.
func effective(keys []Keybinding, command string, ctx map[string]bool) []Keybinding {
	taken := map[string]bool{}
	var out []Keybinding
	for i := len(keys) - 1; i >= 0; i-- {
		k := keys[i]
		if !k.When.holds(ctx) || taken[k.claim()] {
			continue
		}
		taken[k.claim()] = true
		if k.Command == command {
			out = append(out, k)
		}
	}
	return out
}

// KeyLabel is command's shortcut as T3 writes it (shortcutLabelForCommand):
// "Ctrl+Shift+B", or "⇧⌘B" on a Mac; "" when it has none.
func KeyLabel(keys []Keybinding, command string) string {
	ks := effective(keys, command, labelContext)
	if len(ks) == 0 {
		return ""
	}
	k := ks[0]
	name := keyName(k.Key)
	mac := runtime.GOOS == "darwin"
	meta, ctrl := k.Meta || k.Mod && mac, k.Ctrl || k.Mod && !mac
	if mac {
		var b strings.Builder
		for _, m := range []struct {
			on  bool
			sym string
		}{{ctrl, "⌃"}, {k.Alt, "⌥"}, {k.Shift, "⇧"}, {meta, "⌘"}} {
			if m.on {
				b.WriteString(m.sym)
			}
		}
		return b.String() + name
	}
	var parts []string
	for _, m := range []struct {
		on   bool
		word string
	}{{ctrl, "Ctrl"}, {k.Alt, "Alt"}, {k.Shift, "Shift"}, {meta, "Meta"}} {
		if m.on {
			parts = append(parts, m.word)
		}
	}
	return strings.Join(append(parts, name), "+")
}

// withKey is label with its command's shortcut after it in brackets, as
// T3's tooltips put it: "Toggle terminal drawer (Ctrl+J)".
func withKey(label string, keys []Keybinding, command string) string {
	if k := KeyLabel(keys, command); k != "" {
		return label + " (" + k + ")"
	}
	return label
}

// keyName is a key's name in a label (formatShortcutKeyLabel).
func keyName(k string) string {
	switch k {
	case " ":
		return "Space"
	case "escape":
		return "Esc"
	case "arrowup", "arrowdown", "arrowleft", "arrowright":
		k = strings.TrimPrefix(k, "arrow")
	}
	if len([]rune(k)) == 1 {
		return strings.ToUpper(k)
	}
	return strings.ToUpper(k[:1]) + k[1:]
}

// gioKey is the key T3 names k as Gio names it; "" for one it has no name
// for.
func gioKey(k string) key.Name {
	switch k {
	case "escape":
		return key.NameEscape
	case "enter":
		return key.NameReturn
	case "tab":
		return key.NameTab
	case " ", "space":
		return key.NameSpace
	case "backspace":
		return key.NameDeleteBackward
	case "delete":
		return key.NameDeleteForward
	case "arrowup":
		return key.NameUpArrow
	case "arrowdown":
		return key.NameDownArrow
	case "arrowleft":
		return key.NameLeftArrow
	case "arrowright":
		return key.NameRightArrow
	case "home":
		return key.NameHome
	case "end":
		return key.NameEnd
	case "pageup":
		return key.NamePageUp
	case "pagedown":
		return key.NamePageDown
	}
	if len([]rune(k)) == 1 {
		return key.Name(strings.ToUpper(k))
	}
	if n, err := strconv.Atoi(strings.TrimPrefix(k, "f")); err == nil && strings.HasPrefix(k, "f") && n >= 1 && n <= 12 {
		return key.Name("F" + strconv.Itoa(n))
	}
	return ""
}

// keyFilters are Gio filters for the shortcuts of commands that hold in
// ctx, exactly their modifiers.
func keyFilters(keys []Keybinding, commands []string, ctx map[string]bool) []event.Filter {
	var out []event.Filter
	for _, c := range commands {
		for _, k := range effective(keys, c, ctx) {
			if name := gioKey(k.Key); name != "" {
				out = append(out, key.Filter{Name: name, Required: k.mods()})
			}
		}
	}
	return out
}

// keyCommand is the command e runs in ctx (resolveShortcutCommand): the
// last rule that holds and whose key and modifiers e are; "" for none.
func keyCommand(keys []Keybinding, e key.Event, ctx map[string]bool) string {
	for i := len(keys) - 1; i >= 0; i-- {
		k := keys[i]
		if k.When.holds(ctx) && gioKey(k.Key) == e.Name && k.mods() == e.Modifiers {
			return k.Command
		}
	}
	return ""
}
