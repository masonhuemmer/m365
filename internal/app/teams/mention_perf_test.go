//go:build !race

package teams

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

// Not built under -race: the race detector slows this code several-fold, so a
// wall-clock limit there measures the runner, not the code (same rule as
// msgbody/perf_deadline_test.go). These guard against quadratic mention work.

// Many mentions in one long paragraph must stay linear.
func TestMentionManyInOneParagraphIsFast(t *testing.T) {
	st := group()
	body := "<p>" + strings.Repeat("@Bo <strong>x</strong> ", 30000) + "</p>"
	start := time.Now()
	sendHTML(t, st, body)
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("30000 mentions took %v", d)
	}
	if len(st.last.Mentions) != 1 {
		t.Fatalf("mentions %d", len(st.last.Mentions))
	}
}

// Many mentions in one text node, followed by inline content, must stay linear.
func TestMentionManyInOneTextNodeIsFast(t *testing.T) {
	st := group()
	body := "<p>" + strings.Repeat("@Bo ", 30000) + "<strong>x</strong></p>"
	start := time.Now()
	sendHTML(t, st, body)
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("30000 mentions in one node took %v", d)
	}
	if len(st.last.Mentions) != 1 {
		t.Fatalf("mentions %d", len(st.last.Mentions))
	}
}

func TestMentionLookaheadIsBoundedInsideALargeTextToken(t *testing.T) {
	st := group()
	body := "<p>" + strings.Repeat("@Bo<i></i> ", 3000) + "<strong>" + strings.Repeat("x", 262144) + "</strong></p>"
	start := time.Now()
	sendHTML(t, st, body)
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("took %v", d)
	}
}

func TestMentionManyDistinctUnknownHandlesIsFast(t *testing.T) {
	var b strings.Builder
	b.WriteString("<p>")
	for i := 0; i < 40000; i++ {
		b.WriteString("@u" + strconv.Itoa(i) + " ")
	}
	b.WriteString("</p>")
	start := time.Now()
	sendHTML(t, group(), b.String())
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("took %v", d)
	}
}
