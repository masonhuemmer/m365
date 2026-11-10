package auth

import (
	"context"
	"time"

	"github.com/masonhuemmer/m365/internal/domain"
)

type Blob struct {
	Account      string    `json:"account"`
	Mail         bool      `json:"mail"`
	Teams        bool      `json:"teams"`
	Calendar     bool      `json:"calendar"`
	Files        bool      `json:"files"`
	Meetings     bool      `json:"meetings"`
	Usable       bool      `json:"usable"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

type Store interface {
	Get() (Blob, bool, error)
	Put(Blob) error
	Delete() error
}

type LoginFn func(ctx context.Context) (Blob, error)

// BrowserOpener opens an authorize URL (system browser in production, fake in tests).
type BrowserOpener func(authorizeURL string) error

// LoopbackStart binds 127.0.0.1 and returns the redirect URL plus a wait for the code.
type LoopbackStart func() (redirectURL string, wait func(context.Context) (code string, err error), err error)

func Status(s Store) (domain.Session, error) {
	b, ok, err := s.Get()
	if err != nil {
		return domain.Session{}, err
	}
	if !ok {
		return domain.SignedOut(), nil
	}
	return domain.Session{
		SignedIn:          true,
		SessionUsable:     b.Usable,
		Account:           b.Account,
		MailConsented:     b.Mail,
		TeamsConsented:    b.Teams,
		CalendarConsented: b.Calendar,
		FilesConsented:    b.Files,
		MeetingsConsented: b.Meetings,
	}, nil
}

func Logout(s Store) error {
	return s.Delete()
}

func Login(ctx context.Context, s Store, fn LoginFn) (domain.Session, error) {
	if fn == nil {
		return domain.Session{}, domain.Auth("login not available")
	}
	b, err := fn(ctx)
	if err != nil {
		return domain.Session{}, err
	}
	b.Usable = true
	if err := s.Put(b); err != nil {
		return domain.Session{}, err
	}
	return Status(s)
}

func Require(sess domain.Session, mail, teams bool) error {
	if !sess.SignedIn || !sess.SessionUsable {
		return domain.Auth("not signed in")
	}
	if mail && !sess.MailConsented {
		return domain.Auth("missing mail consent")
	}
	if teams && !sess.TeamsConsented {
		return domain.Auth("missing Teams consent")
	}
	return nil
}

func RequireCalendar(sess domain.Session) error {
	if err := Require(sess, false, false); err != nil {
		return err
	}
	if !sess.CalendarConsented {
		return domain.Auth("missing calendar consent")
	}
	return nil
}

func RequireFiles(sess domain.Session) error {
	if err := Require(sess, false, false); err != nil {
		return err
	}
	if !sess.FilesConsented {
		return domain.Auth("missing files consent")
	}
	return nil
}

// RequireMeetings checks the opt-in meetings consent.
func RequireMeetings(sess domain.Session) error {
	if err := Require(sess, false, false); err != nil {
		return err
	}
	if !sess.MeetingsConsented {
		return &domain.Error{Class: domain.ClassAuth, Message: "missing meetings consent", Hint: "run m365 auth login --meetings"}
	}
	return nil
}
