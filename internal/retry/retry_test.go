package retry

import (
	"errors"
	"testing"
	"time"
)

var expectedDelays = []time.Duration{
	250_000_000,
	312_500_000,
	390_625_000,
	488_281_250,
	610_351_562,
	762_939_452,
}

type mockTime struct {
	now    time.Time
	sleeps []time.Duration
}

func newMT(start time.Time) *mockTime {
	return &mockTime{now: start}
}

func (m *mockTime) Now() time.Time {
	return m.now
}

func (m *mockTime) Sleep(d time.Duration) {
	m.sleeps = append(m.sleeps, d)
	m.now = m.now.Add(d)
}

func (m *mockTime) Advance(d time.Duration) {
	m.now = m.now.Add(d)
}

func TestRetrySuccessOnFirstTry(t *testing.T) {
	var mt = newMT(time.Now())

	var callCount = 0
	var fn = func() error {
		callCount++
		return nil
	}

	var err = doWithTimeProvider(time.Second, fn, mt)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
	if len(mt.sleeps) != 0 {
		t.Errorf("Expected no sleep calls, got %v", mt.sleeps)
	}
}

func TestRetrySuccessAfterFailures(t *testing.T) {
	var mt = newMT(time.Now())

	var callCount = 0
	var fn = func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}

	var err = doWithTimeProvider(time.Second, fn, mt)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}

	if len(mt.sleeps) != 2 {
		t.Errorf("Expected 2 sleep calls, got %d", len(mt.sleeps))
	}
	for i, expected := range expectedDelays {
		if i >= 2 {
			break
		}
		if mt.sleeps[i] != expected {
			t.Errorf("Sleep call %d: expected %v, got %v", i, expected, mt.sleeps[i])
		}
	}
}

func TestRetryTimeout(t *testing.T) {
	var mt = newMT(time.Now())
	var maxWait = time.Minute

	var callCount = 0
	var fn = func() error {
		callCount++
		return errors.New("persistent failure")
	}

	var err = doWithTimeProvider(maxWait, fn, mt)
	if err == nil {
		t.Error("Expected error due to timeout, got nil")
	}
	if err.Error() != "persistent failure" {
		t.Errorf("Expected 'persistent failure', got %v", err)
	}

	// Verify that we stopped retrying not too long after maxWait
	var totalSlept = time.Duration(0)
	for _, sleep := range mt.sleeps {
		totalSlept += sleep
	}

	if totalSlept >= (maxWait + MaxDelay) {
		t.Errorf("Total sleep time %v should be maxWait (%v) or slightly after (MaxDelay + maxWait: %v)", totalSlept, maxWait, maxWait+MaxDelay)
	}
}

func TestRetryDelayProgression(t *testing.T) {
	var mt = newMT(time.Now())

	var callCount = 0
	var fn = func() error {
		callCount++
		if callCount <= len(expectedDelays) {
			return errors.New("keep failing")
		}
		return nil
	}

	var err = doWithTimeProvider(time.Minute, fn, mt)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	for i, expected := range expectedDelays {
		if i >= len(mt.sleeps) {
			break
		}
		var actual = mt.sleeps[i]
		if actual != expected {
			t.Errorf("Delay %d: expected %v, got %v", i, expected, actual)
		}
	}
}

func TestRetryMaxDelay(t *testing.T) {
	var mt = newMT(time.Now())

	// Create a scenario where delay would exceed MaxDelay. Just blasting 100
	// retries should do it.
	var callCount = 0
	var fn = func() error {
		callCount++
		if callCount <= 100 {
			return errors.New("keep failing")
		}
		return nil
	}

	var err = doWithTimeProvider(time.Minute*20, fn, mt)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// We just verify max delay was hit at least once
	var foundMaxDelay = false
	for _, sleep := range mt.sleeps {
		if sleep == MaxDelay {
			foundMaxDelay = true
			break
		}
		if sleep > MaxDelay {
			t.Errorf("Sleep duration %v exceeded MaxDelay %v", sleep, MaxDelay)
		}
	}

	if !foundMaxDelay {
		t.Error("Expected to find at least one sleep at MaxDelay")
	}
}

func TestRetryAlwaysCallsAtLeastOnce(t *testing.T) {
	var mt = newMT(time.Now())

	var callCount = 0
	var fn = func() error {
		callCount++
		return errors.New("always fails")
	}

	// Zero maxWait should still call function once
	_ = doWithTimeProvider(0, fn, mt)
	if callCount != 1 {
		t.Errorf("Expected 1 call even with zero maxWait, got %d", callCount)
	}
}

// Integration test with real time (shorter duration)
func TestRetryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	var start = time.Now()
	var callCount = 0

	var fn = func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}

	var err = Do(time.Second, fn)
	var elapsed = time.Since(start)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}

	// Should have taken at least InitialDelay + (InitialDelay * Multiplier)
	var expectedMinTime = InitialDelay + time.Duration(float64(InitialDelay)*Multiplier)
	if elapsed < expectedMinTime {
		t.Errorf("Expected at least %v elapsed time, got %v", expectedMinTime, elapsed)
	}
}
