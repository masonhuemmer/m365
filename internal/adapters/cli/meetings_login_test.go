package cli

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// loginRecorder gives each login function a name and records which ran.
func loginRecorder(d *Deps, meetings bool) *[]string {
	called := &[]string{}
	fn := func(name string) auth.LoginFn {
		return func(context.Context) (auth.Blob, error) {
			*called = append(*called, name)
			return auth.Blob{Account: "user@example.com", Mail: true, Teams: true, Calendar: true, Files: true, Meetings: meetings, Usable: true}, nil
		}
	}
	d.Login, d.DeviceLogin = fn("plain"), fn("device")
	d.MeetingsLogin, d.MeetingsDeviceLogin = fn("meetings"), fn("meetings-device")
	return called
}

func TestAuthLoginPicksTheLoginFunction(t *testing.T) {
	cases := []struct {
		flags []string
		want  string
	}{
		{nil, "plain"},
		{[]string{"--device-code"}, "device"},
		{[]string{"--meetings"}, "meetings"},
		{[]string{"--device-code", "--meetings"}, "meetings-device"},
		{[]string{"--meetings", "--device-code"}, "meetings-device"},
	}
	for _, c := range cases {
		d, _, errw := testDeps()
		called := loginRecorder(&d, true)
		if code := Run(append([]string{"m365", "auth", "login"}, c.flags...), d); code != 0 {
			t.Fatalf("%v: exit %d %s", c.flags, code, errw.String())
		}
		if !reflect.DeepEqual(*called, []string{c.want}) {
			t.Errorf("%v: called %v, want %s", c.flags, *called, c.want)
		}
	}
}

func TestAuthLoginRejectsARepeatedFlag(t *testing.T) {
	d, _, errw := testDeps()
	called := loginRecorder(&d, true)
	if code := Run([]string{"m365", "auth", "login", "--meetings", "--meetings"}, d); code != 3 || len(*called) != 0 {
		t.Fatalf("exit %d called %v", code, *called)
	}
	var e errObj
	_ = json.Unmarshal(errw.Bytes(), &e)
	if e.Class != "usage" || e.Message != `unknown login flag "--meetings"` {
		t.Fatalf("stderr %q", errw.String())
	}
}

func TestAuthLoginMeetingsNotAvailable(t *testing.T) {
	d, _, errw := testDeps()
	if code := Run([]string{"m365", "auth", "login", "--meetings"}, d); code != 4 {
		t.Fatalf("exit %d", code)
	}
	if errw.String() != `{"class":"auth","message":"login not available"}`+"\n" {
		t.Fatalf("stderr %q", errw.String())
	}
}

func statusNamespaces(t *testing.T, d Deps) map[string]any {
	t.Helper()
	_, out, _ := testDeps()
	d.Stdout = out
	if code := Run([]string{"m365", "auth", "status"}, d); code != 0 {
		t.Fatal(code)
	}
	var st map[string]any
	if err := json.Unmarshal(out.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	return st["namespaces"].(map[string]any)
}

func TestAuthStatusReportsMeetings(t *testing.T) {
	d, _, _ := testDeps()
	loginRecorder(&d, true)
	Run([]string{"m365", "auth", "login", "--meetings"}, d)
	want := map[string]any{"mail": true, "teams": true, "calendar": true, "files": true, "meetings": true}
	if got := statusNamespaces(t, d); !reflect.DeepEqual(got, want) {
		t.Fatalf("namespaces %v", got)
	}
	d, _, _ = testDeps()
	loginRecorder(&d, false)
	Run([]string{"m365", "auth", "login"}, d)
	if got := statusNamespaces(t, d); got["meetings"] != false {
		t.Fatalf("namespaces %v", got)
	}
}

func TestMeetingsVerbWithoutMeetingsConsent(t *testing.T) {
	d, _, errw := testDeps()
	d.Meetings = newCLIMeetings(t, inWindow())
	loginRecorder(&d, false)
	Run([]string{"m365", "auth", "login"}, d)
	errw.Reset()
	p := filepath.Join(t.TempDir(), "m.vtt")
	if code := Run([]string{"m365", "meetings", "transcript", "--join-url", "https://teams.example.test/x", "--out", p}, d); code != 4 {
		t.Fatalf("exit %d", code)
	}
	if errw.String() != `{"class":"auth","message":"missing meetings consent","hint":"run m365 auth login --meetings"}`+"\n" {
		t.Fatalf("stderr %q", errw.String())
	}
}

func TestRootHelpNamesTheMeetingsLogin(t *testing.T) {
	d, out, _ := testDeps()
	Run([]string{"m365", "--help"}, d)
	if !strings.Contains(out.String(), "\nSign in: m365 auth login [--device-code] [--meetings]\n") {
		t.Fatalf("root help %q", out.String())
	}
}

func TestMCPStatusShowsMeetings(t *testing.T) {
	d, _, _ := testDeps()
	loginRecorder(&d, true)
	Run([]string{"m365", "auth", "login", "--meetings"}, d)
	res, err := connectMCP(t, d).CallTool(context.Background(), &mcp.CallToolParams{Name: "m365_status"})
	if err != nil || res.IsError {
		t.Fatal(err)
	}
	var st map[string]any
	_ = json.Unmarshal([]byte(toolText(t, res)), &st)
	if ns, _ := st["namespaces"].(map[string]any); ns["meetings"] != true {
		t.Fatalf("status %v", st)
	}
}
