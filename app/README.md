# tern app (Go + Gio)

A native T3 Code client in Go. The UI is [Gio](https://gioui.org), an
immediate-mode GPU toolkit written in pure Go. The Windows build needs no C
toolchain; the Mac build needs one for Gio's Cocoa code.

The interface is a 1:1 port of T3 Code's own client, pinned to the nightly
the server runs. `../docs/ui/` specifies it surface by surface, from T3's
source and measurements of its live web UI (`tools/t3ui/inspect.sh`). The
colours, lengths and type come from `../docs/ui/tokens.json` through
`tools/t3ui/gentokens.py` into `ui/tokens_gen.go`; tern follows the system's
light or dark setting as T3 does. Text is SF Pro and SF Mono on a Mac (loaded
from the system) and T3's own Inter Variable elsewhere, instanced at T3's
weights by `tools/t3ui/instance-inter.py`.

- `t3/` is the protocol: Clerk sign-in, DPoP keys, the relay, environment
  sessions (T3 Connect or a `t3 pair` URL) and the Effect RPC socket. It
  follows `../docs/t3-connect.md`.
- `core/` is the client. It keeps a socket open to every environment, mirrors
  their shells and the open thread, and builds a `ui.Scene` each frame.
- `ui/` draws scenes. The mockups and the app use the same code; a live scene
  turns fields into editors and rows into click targets.
- `cmd/tern` is the app, `cmd/mockup` renders the mockups, and
  `cmd/livecheck` and `cmd/t3probe` test against a real server.

## Build

CI (`.github/workflows/app.yml`) runs gofmt, vet and the tests on Linux,
Windows and macOS on every push, and uploads `tern-windows` (`tern.exe`)
and `tern-mac` (a universal binary, `lipo`'d from arm64 and amd64 and
signed ad hoc) as artifacts. By hand:

```sh
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -H=windowsgui" -o dist/tern.exe ./cmd/tern
go build -o dist/tern-mac ./cmd/tern   # on a Mac
```

On Windows, credentials live in `%AppData%\tern`, sealed with DPAPI so only
the same Windows user can read them. The icon is
`cmd/tern/rsrc_windows_*.syso`; to redraw it, run `go run .` in
`tools/winicon` (its own module, so the app doesn't depend on winres). On a
Mac they live in `~/Library/Application Support/tern` as 0600 files. The Mac
build is a plain binary, not an `.app`: downloaded with curl it carries no
quarantine flag, so Gatekeeper leaves it alone.

## What works

- Sign in with your T3 Connect email and the emailed code (or a password), or
  paste a pairing URL from `t3 pair` for a machine that isn't on Connect.
- Every environment on the account at once, with online status. The inbox
  mixes their threads and tags each with its machine.
- The inbox: Pinned, Active, Snoozed and Settled, with T3's status chips,
  live run timers and title search.
- Threads: messages, reasoning, tool calls (click to load their output),
  plans and errors, with finished runs folded into "Worked for…". "Load
  earlier" pages in older history.
- The composer: Enter sends (the server steers or queues if a run is busy),
  Alt+Enter queues, Shift+Enter is a newline. The model picker, the options
  menu (effort, fast mode, service tier, …) and the access menu come from the
  server's config. Unsent text is kept per thread while the app runs.
- Ctrl+V (⌘V on a Mac) pastes images into the composer as chips at the caret, like the web
  composer's, sent at that spot in the message: screenshots, pictures copied
  from a browser, or image files copied in Explorer or the Finder (up to 8, 10 MB each).
  Backspace or × removes one, undo brings it back. Sent messages show their
  pictures and chips too.
- Right-click a thread for Pin, Settle (or Wake up), Mark unread, Archive
  and Delete; Delete asks for a second click.
- One thread at a time, like T3; the sidebar switches. The header's details
  button (⇧⌘B, Ctrl+Shift+B) drops T3's details popover: machine, run, model,
  access, and the branch and folder, which copy on click. The sidebar's
  footer has a machines button (each machine and its status) and an account
  button (sign out). Icons are T3's lucide set, copied by `tools/lucide`.
- On a Mac the window has no title bar: the traffic lights sit in the top row
  and the headers drag the window.
- New threads in any project on any machine (the dropdown in the header),
  approvals (Approve, For session, Decline), agent questions,
  the queue (send now, cancel) and Stop.
- Reconnects with backoff after a dropped socket or sleep, and renews the
  1-hour Connect session before it runs out.

- The terminal drawer (⌘J, Ctrl+J): shells on the thread's machine in its
  folder, as many as you like, shared with T3's own (the same `term-N` ids).
  The emulator is charmbracelet/x/vt; the wheel scrolls back, drag selects,
  ⌘C (Ctrl+C with a selection) copies, ⌘V pastes. A shell survives a dropped
  connection: its stream resumes when the machine is back. Drag the
  drawer's top edge to resize it.
- The browser panel (the header's right-panel button): a tab in the
  machine's own headless Chromium, streamed as JPEG frames over
  `/api/preview-stream/ws`, with clicks, scrolling, typing and the address
  bar going back to it. Tabs are the thread's in T3 too; a blank one lists
  the dev servers started from T3's terminals there, and `localhost:3000` or
  just `3000` opens one. Drag the panel's left edge to resize it.
- Text falls back to the system's fonts for scripts the embedded ones don't
  have, like Chinese and Japanese.

Not yet: diffs, dropping files on the window, `/`
commands and `@` files, and renaming or snoozing from the thread menu.

## Mockups

```sh
# Linux with no display: Mesa's software EGL, no Vulkan headers needed.
EGL_PLATFORM=surfaceless go run -tags novulkan ./cmd/mockup -out mockups

# The Fast Mode switch animation: frames, then a GIF (needs Pillow).
EGL_PLATFORM=surfaceless go run -tags novulkan ./cmd/mockup -out mockups -anim /tmp/frames
python3 tools/gif.py /tmp/frames/fast-toggle mockups/fast-toggle.gif
```

## Testing against a server

```sh
# On the host: t3 pair --scope orchestration:read --label "tern test"
# Pair a store with the URL it prints, then drive the app headlessly.
go run ./cmd/t3probe -store /tmp/tern-probe -pair "<pairing URL>"
EGL_PLATFORM=surfaceless go run -tags novulkan ./cmd/livecheck -store /tmp/tern-probe
```

`livecheck` clicks through the inbox, a thread, the menus and sign-in, and
writes a screenshot per step. With a read-only session it also checks that
sending is refused, so it never posts anything.

Fonts are Inter and JetBrains Mono (SIL OFL, licenses in `ui/fonts/`),
embedded in the binary.
