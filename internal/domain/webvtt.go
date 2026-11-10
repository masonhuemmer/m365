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

// VoiceNames returns the distinct voice-span names in b (<v Name> or
// <v.class Name>), trimmed, in order of first appearance.
func VoiceNames(b []byte) []string {
	var names []string
	seen := map[string]bool{}
	for _, m := range voiceOpener.FindAllSubmatch(b, -1) {
		name := strings.TrimSpace(string(m[1]))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}
