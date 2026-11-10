package domain

// IsWebVTT reports whether b starts with the WebVTT header.
func IsWebVTT(b []byte) bool {
	return false
}

// VoiceNames returns the distinct voice-span names in b, in order of first appearance.
func VoiceNames(b []byte) []string {
	return nil
}
