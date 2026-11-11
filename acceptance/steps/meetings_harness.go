package steps

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/masonhuemmer/m365/acceptance/runtime"
	"github.com/masonhuemmer/m365/internal/adapters/cli"
	"github.com/masonhuemmer/m365/internal/adapters/graph"
	"github.com/masonhuemmer/m365/internal/adapters/keychain"
	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/config"
)

// meetingsVTT is the synthetic fixture (fictional names), the same bytes as
// internal/domain/testdata/meeting.vtt.
const meetingsVTT = "\xef\xbb\xbfWEBVTT\n\nNOTE synthetic fixture\n\n1\n00:00:01.000 --> 00:00:03.000\n" +
	"<v Avery Example>Opening the review.</v>\n\n00:00:03.500 --> 00:00:06.000 \n" +
	"<v.loud Blake Sample>Two lines here,\nand a second line.</v>\n\n00:00:06.500 --> 00:00:07.000\n(inaudible)\n\n" +
	"00:00:07.500 --> 00:00:09.000\n<v Avery Example>Back to the agenda.</v>\n"

const meetingsJoinURL = "https://teams.example.test/l/meetup-join/abc"

// meetingsAnswer is one canned Graph answer; a zero value means 404.
type meetingsAnswer struct {
	status int
	body   []byte
}

// meetingsFixture is what the TLS fake Graph serves. Scenarios change its
// fields instead of editing this harness.
type meetingsFixture struct {
	events   map[string]meetingsAnswer
	meetings map[string]string // join URL -> meeting id
	pages    [][]map[string]string
	list     *meetingsAnswer // when set, every list request gets this answer
	content  map[string]meetingsAnswer
}

// meetingsRun is one scenario's fake, sign-in and destination.
type meetingsRun struct {
	fx    meetingsFixture
	login auth.Blob
	dest  string
	base  string
}

var meetingsRuns sync.Map

func newMeetingsRun(w *runtime.World) *meetingsRun {
	r := &meetingsRun{
		fx: meetingsFixture{
			events: map[string]meetingsAnswer{"evt-online": {body: []byte(`{"isOnlineMeeting":true,` +
				`"onlineMeeting":{"joinUrl":"` + meetingsJoinURL + `"},` +
				`"start":{"dateTime":"2030-01-15T15:00:00.0000000"},"end":{"dateTime":"2030-01-15T15:30:00.0000000"}}`)}},
			meetings: map[string]string{meetingsJoinURL: "mtg-1"},
			content:  map[string]meetingsAnswer{},
		},
		login: auth.Blob{Account: "user@example.com", Mail: true, Teams: true, Calendar: true, Files: true, Meetings: true, Usable: true},
		dest:  filepath.Join(w.T.TempDir(), "meeting.vtt"),
	}
	meetingsRuns.Store(w, r)
	return r
}

func meetingsRunOf(w *runtime.World) *meetingsRun {
	v, ok := meetingsRuns.Load(w)
	if !ok {
		w.T.Fatal("no meetings fake in this scenario")
	}
	return v.(*meetingsRun)
}

func meetingsTranscript(id, created string) map[string]string {
	return map[string]string{"id": id, "createdDateTime": created, "endDateTime": "2030-01-15T15:29:00Z"}
}

// run signs in with r.login and runs one m365 command against the fake.
func (r *meetingsRun) run(w *runtime.World, args ...string) {
	srv := httptest.NewTLSServer(http.HandlerFunc(r.serve))
	defer srv.Close()
	r.base = srv.URL
	out, errw := &bytes.Buffer{}, &bytes.Buffer{}
	login := r.login
	d := cli.Deps{
		Config:   config.Config{ClientID: "x", TenantID: "y"},
		Store:    &keychain.Fake{},
		Meetings: &graph.HTTPMeetings{HTTPClient: &graph.HTTPClient{Base: srv.URL, Token: "synthetic-token", Client: srv.Client()}},
		Login:    func(context.Context) (auth.Blob, error) { return login, nil },
		Stdout:   out, Stderr: errw,
	}
	if code := cli.Run([]string{"m365", "auth", "login"}, d); code != 0 {
		w.T.Fatalf("login exit %d: %s", code, errw.String())
	}
	out.Reset()
	errw.Reset()
	w.Code = cli.Run(append([]string{"m365"}, args...), d)
	w.Out, w.Err = out.String(), errw.String()
}

func (r *meetingsRun) serve(rw http.ResponseWriter, req *http.Request) {
	p := req.URL.Path
	switch {
	case strings.HasPrefix(p, "/me/events/"):
		meetingsAnswerWith(rw, r.fx.events[strings.TrimPrefix(p, "/me/events/")])
	case p == "/me/onlineMeetings":
		lit := strings.TrimSuffix(strings.TrimPrefix(req.URL.Query().Get("$filter"), "JoinWebUrl eq '"), "'")
		value := []map[string]string{}
		if id, ok := r.fx.meetings[strings.ReplaceAll(lit, "''", "'")]; ok {
			value = append(value, map[string]string{"id": id})
		}
		_ = json.NewEncoder(rw).Encode(map[string]any{"value": value})
	case strings.HasSuffix(p, "/content"):
		parts := strings.Split(p, "/")
		meetingsAnswerWith(rw, r.fx.content[parts[len(parts)-2]])
	case strings.HasSuffix(p, "/transcripts"):
		r.listPage(rw, req)
	default:
		meetingsAnswerWith(rw, meetingsAnswer{})
	}
}

func (r *meetingsRun) listPage(rw http.ResponseWriter, req *http.Request) {
	if r.fx.list != nil {
		meetingsAnswerWith(rw, *r.fx.list)
		return
	}
	n, _ := strconv.Atoi(req.URL.Query().Get("$skiptoken"))
	page := map[string]any{"value": []map[string]string{}}
	if n < len(r.fx.pages) {
		page["value"] = r.fx.pages[n]
	}
	if n+1 < len(r.fx.pages) {
		page["@odata.nextLink"] = r.base + req.URL.Path + "?$skiptoken=" + strconv.Itoa(n+1)
	}
	_ = json.NewEncoder(rw).Encode(page)
}

func meetingsAnswerWith(rw http.ResponseWriter, a meetingsAnswer) {
	if a.status == 0 && a.body == nil {
		a = meetingsAnswer{http.StatusNotFound, []byte(`{"error":{"code":"ErrorItemNotFound"}}`)}
	}
	if a.status == 0 {
		a.status = http.StatusOK
	}
	rw.WriteHeader(a.status)
	_, _ = rw.Write(a.body)
}
