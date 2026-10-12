package graph

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func deviceServer(t *testing.T, tokenBody string) *oauth2.Config {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/devicecode") {
			_, _ = w.Write([]byte(`{"device_code":"d","user_code":"ABCD1234","verification_uri":"https://login.example/device","expires_in":60,"interval":1}`))
			return
		}
		_, _ = w.Write([]byte(tokenBody))
	}))
	t.Cleanup(srv.Close)
	cfg := PKCEConfig("cid", "tid", "")
	cfg.Endpoint.DeviceAuthURL = srv.URL + "/devicecode"
	cfg.Endpoint.TokenURL = srv.URL + "/token"
	return cfg
}

func TestDeviceLoginPromptsAndReturnsSession(t *testing.T) {
	cfg := deviceServer(t, `{"access_token":"tok","token_type":"Bearer","expires_in":3600,"refresh_token":"r","scope":"Mail.Read Chat.Read Files.ReadWrite"}`)
	var out bytes.Buffer
	b, err := deviceLogin(context.Background(), cfg, &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ABCD1234") || !strings.Contains(out.String(), "https://login.example/device") {
		t.Fatal(out.String())
	}
	if b.RefreshToken != "r" || !b.Mail || !b.Teams || !b.Files || b.Calendar {
		t.Fatalf("%+v", b)
	}
}

func TestDeviceLoginDeclined(t *testing.T) {
	cfg := deviceServer(t, `{"error":"authorization_declined"}`)
	if _, err := deviceLogin(context.Background(), cfg, &bytes.Buffer{}); err == nil {
		t.Fatal("expected error")
	}
}
