package t3

import (
	"regexp"
	"strconv"
	"strings"
)

// Composer context, the way T3's web composer sends pictures: the prompt
// carries a markdown token for each one, ![name](t3-context://v1/image/<id>),
// and the message's context carries a record per token with the attachment.
// The server turns the pair into "[Image: name; ref=<id>]" plus a
// <t3_context> block for the agent
// (packages/shared/src/composerContextReferences.ts).

const contextHref = "t3-context://v1/"

// contextLink matches a token: an optional "!", the label, the href.
var contextLink = regexp.MustCompile(`(!?)\[([^\]\n]{0,512})\]\((` + regexp.QuoteMeta(contextHref) + `[^\s)]{1,200})\)`)

// ImageRecord is one picture's context record.
type ImageRecord struct {
	Version      int    `json:"version"`
	Kind         string `json:"kind"`
	ContextID    string `json:"contextId"`
	Label        string `json:"label"`
	AttachmentID string `json:"attachmentId"`
	Name         string `json:"name"`
	MimeType     string `json:"mimeType"`
	SizeBytes    int    `json:"sizeBytes"`
}

// NewImageRecord describes an uploaded picture. id is the client's own id
// for it, a UUID; label is what the prompt calls it ("Image 1").
func NewImageRecord(id, label string, a Attachment) ImageRecord {
	return ImageRecord{Version: 1, Kind: "image", ContextID: "image_" + id, Label: contextLabel(label, "image"),
		AttachmentID: a.ID, Name: a.Name, MimeType: a.MimeType, SizeBytes: a.SizeBytes}
}

// Token is the prompt's reference to r.
func (r ImageRecord) Token() string {
	return "![" + r.Label + "](" + contextHref + r.Kind + "/" + r.ContextID + ")"
}

func contextLabel(label, kind string) string {
	l := strings.Join(strings.Fields(strings.NewReplacer("[", " ", "]", " ", `\`, " ").Replace(label)), " ")
	if len(l) > 200 {
		l = l[:200]
	}
	if l == "" {
		return kind
	}
	return l
}

// WithImages appends the pictures' tokens to text, the way the web places
// pictures that aren't inline.
func WithImages(text string, recs []ImageRecord) string {
	var toks []string
	for _, r := range recs {
		toks = append(toks, r.Token())
	}
	t := strings.Join(toks, " ")
	switch {
	case t == "":
		return text
	case strings.TrimSpace(text) == "":
		return t
	}
	return strings.TrimRight(text, " \t\n") + "\n\n" + t
}

// ProjectImages is the text a server without inlineMessageContext wants:
// what a newer server would hand the agent.
func ProjectImages(text string, recs []ImageRecord) string {
	byID := map[string]ImageRecord{}
	for _, r := range recs {
		byID[r.ContextID] = r
	}
	var entries []string
	seen := map[string]bool{}
	body := contextLink.ReplaceAllStringFunc(text, func(m string) string {
		sub := contextLink.FindStringSubmatch(m)
		kind, id, ok := strings.Cut(strings.TrimPrefix(sub[3], contextHref), "/")
		if !ok {
			return m
		}
		label := strings.Join(strings.Fields(strings.NewReplacer("\r", " ", "\n", " ", ";", " ", "]", " ").Replace(sub[2])), " ")
		if !seen[id] {
			seen[id] = true
			if r, ok := byID[id]; ok {
				entries = append(entries, `<context kind="`+r.Kind+`" id="`+id+`">`+"\n"+
					"name: "+r.Name+"\nmimeType: "+r.MimeType+"\nsizeBytes: "+strconv.Itoa(r.SizeBytes)+"\nattachmentId: "+r.AttachmentID+
					"\n</context>")
			} else {
				entries = append(entries, `<context kind="`+kind+`" id="`+id+`" unavailable="true"/>`)
			}
		}
		return "[" + strings.ToUpper(kind[:1]) + strings.ReplaceAll(kind[1:], "-", " ") + ": " + label + "; ref=" + id + "]"
	})
	if len(entries) == 0 {
		return body
	}
	return body + "\n\n<t3_context version=\"1\">\n" + strings.Join(entries, "\n") + "\n</t3_context>"
}

// ReplaceImages swaps each picture token in text for image(label) and every
// other reference for its label.
func ReplaceImages(text string, image func(label string) string) string {
	if !strings.Contains(text, "]("+contextHref) {
		return text
	}
	return contextLink.ReplaceAllStringFunc(text, func(m string) string {
		sub := contextLink.FindStringSubmatch(m)
		if sub[1] == "!" {
			return image(sub[2])
		}
		return sub[2]
	})
}

// StripContext is text with picture tokens shown as [label] and other
// references as their label.
func StripContext(text string) string {
	return strings.TrimSpace(ReplaceImages(text, func(l string) string { return "[" + l + "]" }))
}
