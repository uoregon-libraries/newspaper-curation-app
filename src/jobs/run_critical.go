package jobs

import (
	"fmt"
	"math"
	"time"
)

// criticalRetries is an internal value for absolutely critical jobs that have
// to succeed (e.g., because of external dependencies which can't be made
// atomic with other internal changes). If this number is exhausted, the job
// fails fatally and cleanup can be a major pain.
const criticalRetries = 20

func (j *Job) runCritical(fn func() error) error {
	var err error
	for n := 0; n < criticalRetries; n++ {
		err = fn()
		if err == nil {
			return nil
		}

		// We do exponential backoff on failures, but *very* slowly - DB issues,
		// the primary use-case for this function, generally resolve quickly
		var millis = int(1000 * math.Pow(1.25, float64(n)))

		// NOTE: I'm well aware that this can't happen with the *current* math
		// we're using. This is for future-proofing in case we make retry count
		// configurable, increase it, or change the exponent above.
		if millis > 120_000 {
			millis = 120_000
		}
		time.Sleep(time.Millisecond * time.Duration(millis))
	}

	return fmt.Errorf("failed after multiple retries: %w", err)
}
