package steps

import (
	"encoding/json"
	"os"
	"reflect"
	goruntime "runtime"
	"strings"

	"github.com/masonhuemmer/m365/acceptance/runtime"
)

func init() {
	runtime.Register("a meetings fake", givenMeetingsFake)
	runtime.Register("the destination file", destinationFile)
	runtime.Register("I save the transcript by", saveTranscript)
	runtime.Register("the saved file", savedFile)
	runtime.Register("the summary", summaryStep)
	runtime.Register("no transcript", noTranscript)
}

// givenMeetingsFake: "a meetings fake with one transcript", "... with two
// transcripts", or "... whose transcript has no speaker names".
func givenMeetingsFake(w *runtime.World, text string) error {
	r := newMeetingsRun(w)
	one := meetingsTranscript("t1", "2030-01-15T15:05:00Z")
	r.fx.pages = [][]map[string]string{{one}}
	r.fx.content["t1"] = meetingsAnswer{body: []byte(meetingsVTT)}
	switch {
	case strings.HasSuffix(text, "with two transcripts"):
		r.fx.pages = [][]map[string]string{{one, meetingsTranscript("t2", "2030-01-15T15:10:00Z")}}
		r.fx.content["t2"] = meetingsAnswer{body: []byte(meetingsVTT)}
	case strings.HasSuffix(text, "has no speaker names"):
		r.fx.content["t1"] = meetingsAnswer{body: []byte("WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nno speaker\n")}
	}
	return nil
}

// destinationFile: "... already exists", "... already exists readable by
// others", or "... is unchanged".
func destinationFile(w *runtime.World, text string) error {
	r := meetingsRunOf(w)
	if strings.HasSuffix(text, "is unchanged") {
		b, err := os.ReadFile(r.dest)
		if err != nil || string(b) != "kept" {
			w.T.Fatalf("destination changed: %q %v", b, err)
		}
		return nil
	}
	if err := os.WriteFile(r.dest, []byte("kept"), 0o600); err != nil {
		return err
	}
	if strings.HasSuffix(text, "readable by others") {
		return os.Chmod(r.dest, 0o644)
	}
	return nil
}

// saveTranscript: "I save the transcript by event|join URL [with id ID|with overwrite]".
func saveTranscript(w *runtime.World, text string) error {
	r := meetingsRunOf(w)
	args := []string{"meetings", "transcript", "--event", "evt-online", "--out", r.dest}
	if strings.Contains(text, "by join URL") {
		args = []string{"meetings", "transcript", "--join-url", meetingsJoinURL, "--out", r.dest}
	}
	if i := strings.Index(text, "with id "); i >= 0 {
		args = append(args, "--id", strings.TrimSpace(text[i+len("with id "):]))
	}
	if strings.HasSuffix(text, "with overwrite") {
		args = append(args, "--overwrite")
	}
	r.run(w, args...)
	return nil
}

// savedFile: "... equals the transcript fixture" or "... is owner-only".
func savedFile(w *runtime.World, text string) error {
	r := meetingsRunOf(w)
	if strings.HasSuffix(text, "is owner-only") {
		st, err := os.Stat(r.dest)
		if err != nil {
			return err
		}
		if goruntime.GOOS != "windows" && st.Mode().Perm() != 0o600 {
			w.T.Fatalf("mode %o, want 600", st.Mode().Perm())
		}
		return nil
	}
	b, err := os.ReadFile(r.dest)
	if err != nil || string(b) != meetingsVTT {
		w.T.Fatalf("saved file differs from the fixture: %v", err)
	}
	return nil
}

// summaryStep: "the summary lists the fixture speakers" or "the summary names transcript ID".
func summaryStep(w *runtime.World, text string) error {
	var got struct {
		ID       string   `json:"id"`
		Speakers []string `json:"speakers"`
	}
	if err := json.Unmarshal([]byte(w.Out), &got); err != nil {
		w.T.Fatalf("stdout is not the summary: %q", w.Out)
	}
	if i := strings.Index(text, "names transcript "); i >= 0 {
		if want := text[i+len("names transcript "):]; got.ID != want {
			w.T.Fatalf("summary id %q, want %q", got.ID, want)
		}
		return nil
	}
	if want := []string{"Avery Example", "Blake Sample"}; !reflect.DeepEqual(got.Speakers, want) {
		w.T.Fatalf("speakers %q, want %q", got.Speakers, want)
	}
	return nil
}

// noTranscript: "no transcript text is printed" or "no transcript file is written".
func noTranscript(w *runtime.World, text string) error {
	if strings.HasSuffix(text, "file is written") {
		if _, err := os.Stat(meetingsRunOf(w).dest); !os.IsNotExist(err) {
			w.T.Fatalf("file written: %v", err)
		}
		return nil
	}
	if strings.Contains(w.Out+w.Err, "Opening the review.") {
		w.T.Fatal("transcript text printed")
	}
	return nil
}
