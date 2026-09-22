package tui

import (
	"testing"
	"time"
)

func TestSetTimeoutReturnsFnResultWhenFast(t *testing.T) {
	got := setTimeout(func() bool { return true }, 50*time.Millisecond, false)
	if !got {
		t.Error("got false, want the fast fn's result (true)")
	}
}

func TestSetTimeoutReturnsFallbackWhenSlow(t *testing.T) {
	duration := 20 * time.Millisecond
	start := time.Now()

	got := setTimeout(func() bool {
		time.Sleep(time.Second)
		return false
	}, duration, true)

	elapsed := time.Since(start)
	if !got {
		t.Error("got false, want the fallback (true)")
	}
	if elapsed > duration+100*time.Millisecond {
		t.Errorf("took %v to return, want close to the %v timeout", elapsed, duration)
	}
}
