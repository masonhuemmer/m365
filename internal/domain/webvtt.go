package domain

import (
	"bytes"
	"regexp"
	"strings"
)

var (
	webVTTBOM   = []byte{0xEF, 0xBB, 0xBF}
	voiceOpener = regexp.MustCompile(`<v(?:\.[^\s>]+)*\s+([^>]+)>`)
)

// IsWebVTT reports whether b starts with the WebVTT header: an optional
// UTF-8 BOM, then WEBVTT, then the end of input, a space, a tab or a newline.
func IsWebVTT(b []byte) bool {
	b = bytes.TrimPrefix(b, webVTTBOM)
	if !bytes.HasPrefix(b, []byte("WEBVTT")) {
		return false
	}
	rest := b[len("WEBVTT"):]
	return len(rest) == 0 || bytes.IndexByte([]byte(" \t\r\n"), rest[0]) >= 0
}

// VoiceNames returns the distinct voice-span names (<v Name> or
// <v.class Name>) in the cue text of b, trimmed, in order of first
// appearance. The header, NOTE, STYLE and REGION blocks and cue ids are
// not cue text.
func VoiceNames(b []byte) []string {
	var names []string
	seen := map[string]bool{}
	for i, block := range webVTTBlocks(b) {
		if i == 0 && bytes.HasPrefix(block, []byte("WEBVTT")) || isNonCueBlock(block) {
			continue
		}
		for _, m := range voiceOpener.FindAllSubmatch(cueText(block), -1) {
			name := strings.TrimSpace(string(m[1]))
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

// webVTTBlocks splits b at blank lines, after dropping a BOM and CRs.
func webVTTBlocks(b []byte) [][]byte {
	b = bytes.ReplaceAll(bytes.TrimPrefix(b, webVTTBOM), []byte("\r\n"), []byte("\n"))
	return bytes.Split(b, []byte("\n\n"))
}

func isNonCueBlock(block []byte) bool {
	first, _, _ := bytes.Cut(block, []byte("\n"))
	for _, kw := range []string{"NOTE", "STYLE", "REGION"} {
		rest, ok := bytes.CutPrefix(first, []byte(kw))
		if ok && (len(rest) == 0 || rest[0] == ' ' || rest[0] == '\t') {
			return true
		}
	}
	return false
}

// cueText drops a cue's id and timing lines; text without a timing line is kept whole.
func cueText(block []byte) []byte {
	if _, after, ok := bytes.Cut(block, []byte("-->")); ok {
		_, text, _ := bytes.Cut(after, []byte("\n"))
		return text
	}
	return block
}
