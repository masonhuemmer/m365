package steps

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/masonhuemmer/m365/acceptance/runtime"
	"github.com/masonhuemmer/m365/internal/adapters/cli"
	"github.com/masonhuemmer/m365/internal/adapters/keychain"
	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/config"
)

func init() {
	runtime.Register("I sign in for meetings", signInForMeetings)
	runtime.Register("auth status reports meetings", authStatusReportsMeetings)
	runtime.Register("a meetings fake without meetings consent", givenNoMeetingsConsent)
}

// signInForMeetings: "I sign in for meetings with a plain login" or "... with the meetings login".
// Each login function grants meetings only when it is the meetings one.
func signInForMeetings(w *runtime.World, text string) error {
	out, errw := &bytes.Buffer{}, &bytes.Buffer{}
	blob := func(meetings bool) auth.LoginFn {
		return func(context.Context) (auth.Blob, error) {
			return auth.Blob{Account: "user@example.com", Mail: true, Teams: true, Meetings: meetings, Usable: true}, nil
		}
	}
	d := cli.Deps{
		Config: config.Config{ClientID: "x", TenantID: "y"}, Store: &keychain.Fake{},
		Login: blob(false), MeetingsLogin: blob(true), Stdout: out, Stderr: errw,
	}
	args := []string{"m365", "auth", "login"}
	if strings.HasSuffix(text, "with the meetings login") {
		args = append(args, "--meetings")
	}
	if code := cli.Run(args, d); code != 0 {
		w.T.Fatalf("login exit %d: %s", code, errw.String())
	}
	out.Reset()
	w.Code = cli.Run([]string{"m365", "auth", "status"}, d)
	w.Out, w.Err = out.String(), errw.String()
	return nil
}

// authStatusReportsMeetings: "auth status reports meetings true|false".
func authStatusReportsMeetings(w *runtime.World, text string) error {
	var st struct {
		Namespaces map[string]bool `json:"namespaces"`
	}
	if err := json.Unmarshal([]byte(w.Out), &st); err != nil {
		w.T.Fatalf("stdout is not auth status: %q", w.Out)
	}
	if want := strings.HasSuffix(text, "true"); st.Namespaces["meetings"] != want {
		w.T.Fatalf("meetings %v, want %v", st.Namespaces["meetings"], want)
	}
	return nil
}

// givenNoMeetingsConsent sets up the save fake with a sign-in that lacks meetings consent.
func givenNoMeetingsConsent(w *runtime.World, _ string) error {
	r := newMeetingsRun(w)
	r.login.Meetings = false
	r.fx.pages = [][]map[string]string{{meetingsTranscript("t1", "2030-01-15T15:05:00Z")}}
	r.fx.content["t1"] = meetingsAnswer{body: []byte(meetingsVTT)}
	return nil
}
