package graph

import (
	"reflect"
	"testing"
)

func TestBrowserCommandPerPlatform(t *testing.T) {
	const u = "https://login.example/authorize?a=1&b=2"
	cases := map[string]struct {
		name string
		args []string
	}{
		"windows": {"rundll32", []string{"url.dll,FileProtocolHandler", u}},
		"darwin":  {"open", []string{u}},
		"linux":   {"xdg-open", []string{u}},
	}
	for goos, want := range cases {
		name, args := browserCommand(goos, u)
		if name != want.name || !reflect.DeepEqual(args, want.args) {
			t.Fatalf("%s: got %s %v", goos, name, args)
		}
	}
}
