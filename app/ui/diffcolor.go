package ui

import "image/color"

// diffColors is the code view's palette: Pierre's colours as T3's Diff
// panel paints them on --background in red & green, the default
// (docs/ui/13-diff.md §3.3-§3.8). Pierre mixes them from the code
// background, the code foreground and the syntax theme's git colours
// (§3.14); these are the measured results. Counts, numbers and bars take
// the syntax theme's git colours, not T3's --diff-* tokens.
type diffColors struct {
	bg, text, number           color.NRGBA // context rows: background, code, line numbers
	addRow, addNumBg, add      color.NRGBA // additions: row, number cell, base (numbers, bar, header +)
	delRow, delNumBg, del      color.NRGBA // deletions
	delDash                    color.NRGBA // the deletion bar's gaps
	addWord, delWord           color.NRGBA // word emphasis, translucent over the row
	sepBg, sepText, expandHot  color.NRGBA // hunk separators, and an expand button's hover
	selRow, selNumBg, selNum   color.NRGBA // selected lines
	selBar                     color.NRGBA
	bufferGutter               color.NRGBA // split view's gutter beside a missing line
	edge                       color.NRGBA // a hovered header's 3dp left edge, translucent
	title, titleHot            color.NRGBA // the file name, and on hover
	modified, added, deleted   color.NRGBA // change icons
	treeModified, treeAdded    color.NRGBA // the file tree's git status colours
	treeDeleted, treeRenamed   color.NRGBA
	treeHover, treeSel, treeFg color.NRGBA // tree rows, translucent; and its chevrons and guides
}

var diffLight = diffColors{
	bg: rgb(0xfcfcfc), text: rgb(0x0a0a0a), number: rgb(0x565656),
	addRow: rgb(0xecf9f4), addNumBg: rgb(0xeef9f5), add: rgb(0x18a46c),
	delRow: rgb(0xfef0f0), delNumBg: rgb(0xfef1f0), del: rgb(0xd52c36), delDash: rgb(0xfde6e2),
	addWord: argb(0x2618a46c), delWord: argb(0x26d52c36),
	sepBg: rgb(0xf5f5f6), sepText: rgb(0x8a8a8c), expandHot: rgb(0xeaeaea),
	selRow: rgb(0xeff7fd), selNumBg: rgb(0xf0f7fd), selNum: rgb(0x20649e), selBar: rgb(0x009fff),
	bufferGutter: rgb(0xf8f8f8),
	edge:         argb(0x3d18181b), title: rgb(0x18181b), titleHot: rgb(0x182139),
	modified: rgb(0x425cb1), added: rgb(0x007a55), deleted: rgb(0xc10007),
	treeModified: rgb(0x1ca1c7), treeAdded: rgb(0x16a994), treeDeleted: rgb(0xff2e3f), treeRenamed: rgb(0xd5a910),
	treeHover: argb(0x14009fff), treeSel: argb(0x1f18181b), treeFg: rgb(0x84848a),
}

var diffDark = diffColors{
	bg: rgb(0x000000), text: rgb(0xfafafa), number: rgb(0x9b9b9b),
	addRow: rgb(0x02110b), addNumBg: rgb(0x04110b), add: rgb(0x07c480),
	delRow: rgb(0x170705), delNumBg: rgb(0x170806), del: rgb(0xff2e3f), delDash: rgb(0x331512),
	addWord: argb(0x3307c480), delWord: argb(0x33ff2e3f),
	sepBg: rgb(0x070707), sepText: rgb(0x7a7a7a), expandHot: rgb(0x141414),
	selRow: rgb(0x060e15), selNumBg: rgb(0x080e15), selNum: rgb(0x72b6ff), selBar: rgb(0x009fff),
	bufferGutter: rgb(0x0c0c0c),
	edge:         argb(0x3df5f5f5), title: rgb(0xf5f5f5), titleHot: rgb(0xd6dff4),
	modified: argb(0xa650a2ff), added: rgb(0x00d492), deleted: rgb(0xff2e3f),
	treeModified: rgb(0x08c0ef), treeAdded: rgb(0x00cab1), treeDeleted: rgb(0xff6762), treeRenamed: rgb(0xffd452),
	treeHover: argb(0x1a009fff), treeSel: argb(0x1ff5f5f5), treeFg: rgb(0x84848a),
}

// dc is the code view's palette in the current theme.
func dc() *diffColors {
	if dark {
		return &diffDark
	}
	return &diffLight
}

// twAlpha is Tailwind's token/pct: the token with its own alpha times pct%,
// rounded as the browser's color-mix is.
func twAlpha(c color.NRGBA, pct int) color.NRGBA {
	c.A = uint8((int(c.A)*pct + 50) / 100)
	return c
}

// argb is a colour written 0xAARRGGBB.
func argb(c uint32) color.NRGBA {
	return color.NRGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: uint8(c >> 24)}
}
