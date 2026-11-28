// Package scheduleracc_test holds the acceptance tests of the Scheduler integration's resource and
// data source. They need a Home Assistant with that custom integration installed, so they have a
// package, and a container, of their own. The tests for a Home Assistant without the integration
// are with the other resources and data sources.
package scheduleracc_test

import (
	"context"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/toelke/terraform-provider-homeassistant/internal/acctest"
	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

func TestMain(m *testing.M) {
	acctest.Install(acctest.Scheduler)
	acctest.Main(m)
}

// preCheck starts Home Assistant and makes sure the Scheduler integration is set up. A test may
// remove its config entry, so this checks before every test.
func preCheck(t *testing.T) {
	t.Helper()
	acctest.PreCheck(t)
	ctx := context.Background()
	c := haClient(t)
	entries, err := client.NewConfigEntries(c.WS, c.REST).List(ctx, "scheduler")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) > 0 {
		return
	}
	if _, err := client.NewConfigFlows(c.REST).Run(ctx, "scheduler", "", nil); err != nil {
		t.Fatalf("set up the Scheduler integration: %v", err)
	}
}

// removeSchedulerEntries deletes the Scheduler integration's config entries, with its schedules.
func removeSchedulerEntries(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	entries := client.NewConfigEntries(haClient(t).WS, haClient(t).REST)
	all, err := entries.List(ctx, "scheduler")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range all {
		if _, err := entries.Delete(ctx, e.EntryID); err != nil {
			t.Fatal(err)
		}
	}
}

var (
	clientOnce sync.Once
	shared     *client.HAClient
)

// haClient returns one client for the shared instance, so polling does not open a WebSocket per
// call.
func haClient(t *testing.T) *client.HAClient {
	t.Helper()
	ha := acctest.SharedInstance(t)
	clientOnce.Do(func() {
		u, err := url.Parse(ha.URL)
		if err != nil {
			t.Fatal(err)
		}
		shared = client.New(client.Config{URL: u, Token: ha.Token, Timeout: 30 * time.Second})
	})
	return shared
}

func schedulers(t *testing.T) client.Schedulers {
	t.Helper()
	return client.NewSchedulers(haClient(t).WS)
}

// scheduleNamed returns the schedule with the given name, if there is one.
func scheduleNamed(t *testing.T, name string) (client.SchedulerSchedule, bool) {
	t.Helper()
	all, err := schedulers(t).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range all {
		if s.Name != nil && *s.Name == name {
			return s, true
		}
	}
	return client.SchedulerSchedule{}, false
}
