package graph

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeResp is one canned Graph answer. A zero value means 404.
type fakeResp struct {
	status int
	body   []byte
}

type seenReq struct{ path, query, prefer string }

// meetingsFake is a TLS fake Graph for the meetings adapter. Continuation
// links must be https, so plain httptest servers cannot serve them.
type meetingsFake struct {
	events   map[string]fakeResp
	meetings map[string]string // join URL -> meeting id
	pages    [][]map[string]string
	list     *fakeResp // when set, every list request gets this answer
	content  map[string]fakeResp

	mu   sync.Mutex
	seen []seenReq
	srv  *httptest.Server
}

func startMeetingsFake(t *testing.T, f *meetingsFake) *HTTPMeetings {
	t.Helper()
	f.srv = httptest.NewTLSServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return &HTTPMeetings{HTTPClient: &HTTPClient{Base: f.srv.URL, Token: "synthetic-token", Client: f.srv.Client()}}
}

func (f *meetingsFake) requests() []seenReq {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]seenReq(nil), f.seen...)
}

func (f *meetingsFake) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.seen = append(f.seen, seenReq{r.URL.Path, r.URL.RawQuery, r.Header.Get("Prefer")})
	f.mu.Unlock()
	p := r.URL.Path
	switch {
	case strings.HasPrefix(p, "/me/events/"):
		answer(w, f.events[strings.TrimPrefix(p, "/me/events/")])
	case p == "/me/onlineMeetings":
		f.lookup(w, r)
	case strings.HasSuffix(p, "/content"):
		parts := strings.Split(p, "/")
		answer(w, f.content[parts[len(parts)-2]])
	case strings.HasSuffix(p, "/transcripts"):
		f.listPage(w, r)
	default:
		answer(w, fakeResp{})
	}
}

func answer(w http.ResponseWriter, r fakeResp) {
	if r.status == 0 && r.body == nil {
		r = fakeResp{http.StatusNotFound, []byte(`{"error":{"code":"ErrorItemNotFound"}}`)}
	}
	if r.status == 0 {
		r.status = http.StatusOK
	}
	w.WriteHeader(r.status)
	_, _ = w.Write(r.body)
}

func (f *meetingsFake) lookup(w http.ResponseWriter, r *http.Request) {
	lit := strings.TrimSuffix(strings.TrimPrefix(r.URL.Query().Get("$filter"), "JoinWebUrl eq '"), "'")
	value := []map[string]string{}
	if id, ok := f.meetings[strings.ReplaceAll(lit, "''", "'")]; ok {
		value = append(value, map[string]string{"id": id})
	}
	writeJSON(w, map[string]any{"value": value})
}

func (f *meetingsFake) listPage(w http.ResponseWriter, r *http.Request) {
	if f.list != nil {
		answer(w, *f.list)
		return
	}
	n, _ := strconv.Atoi(r.URL.Query().Get("$skiptoken"))
	page := map[string]any{"value": []map[string]string{}}
	if n < len(f.pages) {
		page["value"] = f.pages[n]
	}
	if n+1 < len(f.pages) {
		page["@odata.nextLink"] = f.srv.URL + r.URL.Path + "?$skiptoken=" + strconv.Itoa(n+1)
	}
	writeJSON(w, page)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func transcriptJSON(id, created string) map[string]string {
	return map[string]string{"id": id, "createdDateTime": created, "endDateTime": "2030-01-15T15:29:00Z"}
}

func onlineEvent() fakeResp {
	return fakeResp{body: []byte(`{"isOnlineMeeting":true,"onlineMeeting":{"joinUrl":"https://teams.example.test/l/meetup-join/abc"},` +
		`"start":{"dateTime":"2030-01-15T15:00:00.0000000","timeZone":"UTC"},"end":{"dateTime":"2030-01-15T15:30:00.0000000","timeZone":"UTC"}}`)}
}

// emptyPages returns n list pages; the last one holds the given items.
func emptyPages(n int, last ...map[string]string) [][]map[string]string {
	pages := make([][]map[string]string, n)
	for i := range pages {
		pages[i] = []map[string]string{}
	}
	pages[n-1] = last
	return pages
}
