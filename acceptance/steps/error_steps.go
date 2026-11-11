package steps

import (
	"encoding/json"
	"strings"

	"github.com/masonhuemmer/m365/acceptance/runtime"
)

func init() {
	runtime.Register("the error is", theErrorIs)
}

// theErrorIs: "the error is CLASS: MESSAGE". Stderr holds one JSON error
// with exactly that class and message.
func theErrorIs(w *runtime.World, text string) error {
	class, msg, ok := strings.Cut(strings.TrimPrefix(text, "the error is "), ": ")
	if !ok {
		w.T.Fatalf("bad step %q: want \"the error is CLASS: MESSAGE\"", text)
	}
	var got struct {
		Class   string `json:"class"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(w.Err)), &got); err != nil {
		w.T.Fatalf("stderr is not one JSON error: %q", w.Err)
	}
	if got.Class != class || got.Message != msg {
		w.T.Fatalf("error %s: %s, want %s: %s", got.Class, got.Message, class, msg)
	}
	return nil
}
