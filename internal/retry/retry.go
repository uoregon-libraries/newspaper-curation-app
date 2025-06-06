// Package retry exposes a single function meant for retrying a critical
// operation in scenarios where errors are rare and usually self-recover fairly
// quickly. Databases being "down" due to network blips, for instance, or a
// network filesystem rebooting.
package retry

import (
	"time"
)

const (
	// InitialDelay is how long we wait on the first retry
	InitialDelay = time.Millisecond * 250

	// MaxDelay is the longest interval between retries
	MaxDelay = 10 * time.Second

	// Multiplier is how much each failure increases the delay between tries
	Multiplier = 1.25
)

// timeProvider allows for dependency injection of time functions
type timeProvider interface {
	Now() time.Time
	Sleep(time.Duration)
}

// realTimeProvider implements TimeProvider using real time functions
type realTimeProvider struct{}

func (realTimeProvider) Now() time.Time        { return time.Now() }
func (realTimeProvider) Sleep(d time.Duration) { time.Sleep(d) }

var defaultTimeProvider timeProvider = realTimeProvider{}

// Do attempts to run the given function repeatedly until it succeeds or
// maxWait has expired. Starting at [InitialDelay], each failure increases the
// delay between it and the next try by a factor of [Multiplier], up to a
// maximum of [MaxDelay]. Only the last error is returned. No matter what n is
// set to, the function will always be called at least once, making this safe
// to use with n=0.
//
// This should be used with caution. It is possible to have a network operation
// (which is basically any DB operation) succeed on the server side, but fail
// before the server is able to respond. This is likely very rare, but it can
// happen, and if it does, a retry will attempt to do the same thing that was
// already done. Only use retry.Do when the operation is idempotent or the risk
// of failure is worse than the damage done by repeating an operation.
func Do(maxWait time.Duration, fn func() error) error {
	return doWithTimeProvider(maxWait, fn, defaultTimeProvider)
}

// doWithTimeProvider is the same as Do but allows injecting a custom time provider for testing
func doWithTimeProvider(maxWait time.Duration, fn func() error, tp timeProvider) error {
	var err error
	var delay = InitialDelay
	var start = tp.Now()

	for tp.Now().Sub(start) < maxWait {
		err = fn()
		if err == nil {
			return nil
		}

		tp.Sleep(delay)
		delay = time.Duration(float64(delay) * Multiplier)
		if delay > MaxDelay {
			delay = MaxDelay
		}
	}
	return fn()
}
