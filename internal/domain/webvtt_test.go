package domain

import (
	"os"
	"reflect"
	"testing"
)

func TestIsWebVTT(t *testing.T) {
	yes := []string{"WEBVTT", "WEBVTT\n", "WEBVTT \n", "WEBVTT\t", "WEBVTT\r\n", "\xEF\xBB\xBFWEBVTT\n"}
	for _, s := range yes {
		if !IsWebVTT([]byte(s)) {
			t.Errorf("IsWebVTT(%q) = false, want true", s)
		}
	}
	no := []string{"", "WEBVTTX", " WEBVTT", `{"value":[]}`, "webvtt\n"}
	for _, s := range no {
		if IsWebVTT([]byte(s)) {
			t.Errorf("IsWebVTT(%q) = true, want false", s)
		}
	}
}

func TestVoiceNamesFromFixture(t *testing.T) {
	b, err := os.ReadFile("testdata/meeting.vtt")
	if err != nil {
		t.Fatal(err)
	}
	got := VoiceNames(b)
	want := []string{"Avery Example", "Blake Sample"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("VoiceNames = %q, want %q", got, want)
	}
}

func TestVoiceNamesEdgeCases(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"<v   Pat Doe  >x</v>", []string{"Pat Doe"}},
		{"<v.a.b Pat Doe>x</v>", []string{"Pat Doe"}},
		{"<v >x</v>", nil},
		{"<v>x</v>", nil},
		{"<b>x</b> <i>y</i>", nil},
	}
	for _, c := range cases {
		got := VoiceNames([]byte(c.in))
		if len(c.want) == 0 {
			if len(got) != 0 {
				t.Errorf("VoiceNames(%q) = %q, want none", c.in, got)
			}
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("VoiceNames(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
