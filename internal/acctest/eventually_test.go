package acctest

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestEventuallyRetriesErrors(t *testing.T) {
	calls := 0
	Eventually(t, time.Second, "third call", func() (bool, error) {
		calls++
		if calls < 3 {
			return false, errors.New("not yet")
		}
		return true, nil
	})
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

// fatalTB records Fatal instead of stopping the test.
type fatalTB struct {
	testing.TB
	msg string
}

func (f *fatalTB) Helper() {}

func (f *fatalTB) Fatal(args ...any) { f.msg = fmt.Sprint(args...) }

func TestEventuallyTimesOut(t *testing.T) {
	tb := &fatalTB{TB: t}
	Eventually(tb, 0, "the moon", func() (bool, error) { return false, errors.New("eclipse") })
	if want := "the moon: not within 0s, last error: eclipse"; tb.msg != want {
		t.Errorf("message = %q, want %q", tb.msg, want)
	}
}
