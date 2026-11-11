package domain

import (
	"reflect"
	"testing"
)

func TestVoiceNamesReadsCueTextOnly(t *testing.T) {
	cases := []struct {
		name, in string
		want     []string
	}{
		{"voice tag in a NOTE block", "WEBVTT\n\nNOTE <v Confidential note text>\n\n00:00:01.000 --> 00:00:02.000\nUnattributed speech\n", nil},
		{"voice tag in the header", "WEBVTT <v Header Name>\n\n00:00:01.000 --> 00:00:02.000\n<v Real Name>x</v>\n", []string{"Real Name"}},
		{"STYLE and REGION blocks", "WEBVTT\n\nSTYLE\n<v Style Name>\n\nREGION\n<v Region Name>\n\n00:00:01.000 --> 00:00:02.000\n<v Real Name>x</v>\n", []string{"Real Name"}},
		{"voice tag in a cue id", "WEBVTT\n\n<v Id Name>\n00:00:01.000 --> 00:00:02.000\n<v Real Name>x</v>\n", []string{"Real Name"}},
		{"CRLF line endings", "WEBVTT\r\n\r\nNOTE <v Hidden>\r\n\r\n00:00:01.000 --> 00:00:02.000\r\n<v Real Name>x</v>\r\n", []string{"Real Name"}},
	}
	for _, c := range cases {
		got := VoiceNames([]byte(c.in))
		if len(c.want) == 0 && len(got) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: VoiceNames = %q, want %q", c.name, got, c.want)
		}
	}
}
