package auth

import (
	"testing"
	"time"
)

func TestStreamTimeoutsSetAndGet(t *testing.T) {
	defer SetStreamTimeouts(DefaultChatHeaderTimeout, DefaultChatIdleTimeout)
	got := CurrentStreamTimeouts()
	if got.Header != DefaultChatHeaderTimeout || got.Idle != DefaultChatIdleTimeout {
		t.Fatalf("expected defaults, got %+v", got)
	}
	// Zero values normalize to the defaults instead of disabling the timeout.
	SetStreamTimeouts(3*time.Second, 0)
	got = CurrentStreamTimeouts()
	if got.Header != 3*time.Second || got.Idle != DefaultChatIdleTimeout {
		t.Fatalf("expected {3s default}, got %+v", got)
	}
}
