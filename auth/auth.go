// Package auth holds shared stream timeout settings.
package auth

import (
	"sync/atomic"
	"time"
)

const (
	DefaultChatHeaderTimeout = 120 * time.Second
	DefaultChatIdleTimeout   = 300 * time.Second
)

type StreamTimeouts struct {
	Header time.Duration
	Idle   time.Duration
}

var timeouts atomic.Pointer[StreamTimeouts]

func init() { SetStreamTimeouts(DefaultChatHeaderTimeout, DefaultChatIdleTimeout) }
func SetStreamTimeouts(header, idle time.Duration) {
	if header <= 0 {
		header = DefaultChatHeaderTimeout
	}
	if idle <= 0 {
		idle = DefaultChatIdleTimeout
	}
	timeouts.Store(&StreamTimeouts{Header: header, Idle: idle})
}
func CurrentStreamTimeouts() StreamTimeouts { return *timeouts.Load() }
