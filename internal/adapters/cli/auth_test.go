package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/domain"
)

func TestAuthStatusSignedOutLoginLogout(t *testing.T) {
	d, out, errw := testDeps()
	code := Run([]string{"m365", "auth", "status"}, d)
	if code != 0 {
		t.Fatal(errw.String())
	}
	var st map[string]any
	if err := json.Unmarshal(out.Bytes(), &st); err != nil {
		t.Fatal(err, out.String())
	}
	if st["signed_in"] != false {
		t.Fatal(st)
	}
	if strings.Contains(out.String(), "token") {
		t.Fatal("token leaked")
	}
	out.Reset()
	code = Run([]string{"m365", "auth", "login"}, d)
	if code != 0 {
		t.Fatal(errw.String())
	}
	out.Reset()
	_ = Run([]string{"m365", "auth", "status"}, d)
	_ = json.Unmarshal(out.Bytes(), &st)
	if st["signed_in"] != true {
		t.Fatal(st)
	}
	out.Reset()
	if c := Run([]string{"m365", "auth", "logout"}, d); c != 0 {
		t.Fatal(c)
	}
	out.Reset()
	_ = Run([]string{"m365", "auth", "status"}, d)
	_ = json.Unmarshal(out.Bytes(), &st)
	if st["signed_in"] != false {
		t.Fatal(st)
	}
	if c := Run([]string{"m365", "mail", "list"}, d); c != domain.ExitAuth {
		t.Fatal(c, errw.String())
	}
}

func TestAuthLoginDeviceCodeFlag(t *testing.T) {
	d, _, errw := testDeps()
	called := false
	d.DeviceLogin = func(ctx context.Context) (auth.Blob, error) {
		called = true
		return auth.Blob{Mail: true}, nil
	}
	if code := Run([]string{"m365", "auth", "login", "--device-code"}, d); code != 0 || !called {
		t.Fatal(code, called, errw.String())
	}
	if code := Run([]string{"m365", "auth", "login", "--bogus"}, d); code == 0 {
		t.Fatal("unknown login flag accepted")
	}
}
