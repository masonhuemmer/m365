package auth

import (
	"encoding/json"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

type blobStore struct{ b Blob }

func (s *blobStore) Get() (Blob, bool, error) { return s.b, true, nil }
func (s *blobStore) Put(b Blob) error         { s.b = b; return nil }
func (s *blobStore) Delete() error            { return nil }

func TestStatusMapsMeetingsConsent(t *testing.T) {
	st, err := Status(&blobStore{b: Blob{Usable: true, Meetings: true}})
	if err != nil {
		t.Fatal(err)
	}
	if !st.MeetingsConsented {
		t.Fatal("MeetingsConsented = false, want true")
	}
}

func TestOlderBlobHasNoMeetingsConsent(t *testing.T) {
	var b Blob
	if err := json.Unmarshal([]byte(`{"account":"a","usable":true,"mail":true}`), &b); err != nil {
		t.Fatal(err)
	}
	if b.Meetings {
		t.Fatal("Meetings = true for a blob without the key")
	}
}

func TestRequireMeetings(t *testing.T) {
	wantErr := func(err error, msg, hint string) {
		t.Helper()
		de, ok := err.(*domain.Error)
		if !ok || de.Class != domain.ClassAuth || de.Message != msg || de.Hint != hint {
			t.Fatalf("err %#v, want auth %q hint %q", err, msg, hint)
		}
	}
	wantErr(RequireMeetings(domain.Session{}), "not signed in", "")
	signedIn := domain.Session{SignedIn: true, SessionUsable: true, MailConsented: true, TeamsConsented: true}
	wantErr(RequireMeetings(signedIn), "missing meetings consent", "run m365 auth login --meetings")
	signedIn.MeetingsConsented = true
	if err := RequireMeetings(signedIn); err != nil {
		t.Fatalf("err %v, want nil", err)
	}
}
