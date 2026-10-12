package graph

import (
	"context"
	"net/http"
	"net/http/httptest"

	"golang.org/x/oauth2"

	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/domain"
)

func FakeLogin(mail, teams bool) auth.LoginFn {
	return FakeLoginAll(mail, teams, false, false)
}

func FakeLoginAll(mail, teams, calendar, files bool) auth.LoginFn {
	return func(context.Context) (auth.Blob, error) {
		return auth.Blob{Account: "user@example.com", Mail: mail, Teams: teams, Calendar: calendar, Files: files, Usable: true}, nil
	}
}

func PKCEConfig(clientID, tenant, redirect string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:    clientID,
		RedirectURL: redirect,
		Endpoint: oauth2.Endpoint{
			AuthURL:       "https://login.microsoftonline.com/" + tenant + "/oauth2/v2.0/authorize",
			TokenURL:      "https://login.microsoftonline.com/" + tenant + "/oauth2/v2.0/token",
			AuthStyle:     oauth2.AuthStyleInParams,
			DeviceAuthURL: "https://login.microsoftonline.com/" + tenant + "/oauth2/v2.0/devicecode",
		},
		Scopes: []string{"openid", "offline_access", "profile", "Mail.Read", "Mail.Send", "Chat.Read", "ChatMessage.Send", "Calendars.ReadWrite", "Files.ReadWrite"},
	}
}

func LoginError(err error) auth.LoginFn {
	return func(context.Context) (auth.Blob, error) { return auth.Blob{}, err }
}

func TestTokenServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"fake-both","token_type":"Bearer","expires_in":3600,"refresh_token":"r"}`))
	}))
}

func Denied() error { return domain.Auth("login refused") }
