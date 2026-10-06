package acctest

import (
	"fmt"
	"testing"
	"time"
)

// pollInterval is how often Eventually checks its condition.
const pollInterval = 200 * time.Millisecond

// Eventually calls cond until it returns true, and fails the test if it has not by timeout. HA
// applies some changes shortly after the command that causes them returns, e.g. a new config
// entry's devices and entities. An error from cond counts as "not yet"; the last one is in the
// failure message.
func Eventually(t testing.TB, timeout time.Duration, what string, cond func() (bool, error)) {
	t.Helper()
	var err error
	for deadline := time.Now().Add(timeout); ; time.Sleep(pollInterval) {
		var ok bool
		if ok, err = cond(); ok && err == nil {
			return
		}
		if time.Now().After(deadline) {
			break
		}
	}
	msg := fmt.Sprintf("%s: not within %s", what, timeout)
	if err != nil {
		msg += ", last error: " + err.Error()
	}
	t.Fatal(msg)
}
