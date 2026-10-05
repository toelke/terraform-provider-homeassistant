package resources

import (
	"context"
	"time"
)

// pollInterval is the first wait between polls. It doubles up to maxPollInterval. Tests shorten
// it.
var (
	pollInterval    = 250 * time.Millisecond
	maxPollInterval = 5 * time.Second
)

// waitFor calls poll with backoff until it reports done, returns an error, or ctx ends. On ctx's
// end it returns ctx's error.
func waitFor(ctx context.Context, poll func(context.Context) (done bool, err error)) error {
	interval := pollInterval
	for {
		done, err := poll(ctx)
		if err != nil || done {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
		interval = min(2*interval, maxPollInterval)
	}
}
