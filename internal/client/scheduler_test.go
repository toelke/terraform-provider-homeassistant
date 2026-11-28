package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// fakeSchedulerWS serves `scheduler` and `scheduler.add` from memory. Add creates the IDs in
// added, plus the ones in alsoAdded, as if someone else created a schedule at the same moment.
func fakeSchedulerWS(existing []string, added, alsoAdded []string) (fakeWS, *[]map[string]any) {
	ids := append([]string(nil), existing...)
	var calls []map[string]any
	return fakeWS{
		"scheduler": func(map[string]any) (any, error) {
			all := make([]map[string]any, len(ids))
			for i, id := range ids {
				all[i] = map[string]any{"schedule_id": id, "entity_id": "switch.schedule_" + id}
			}
			return all, nil
		},
		"call_service": func(p map[string]any) (any, error) {
			calls = append(calls, p)
			if p["domain"] == "scheduler" && p["service"] == "add" {
				ids = append(ids, added...)
				ids = append(ids, alsoAdded...)
			}
			return nil, nil
		},
	}, &calls
}

func TestSchedulersAdd(t *testing.T) {
	ws, calls := fakeSchedulerWS([]string{"aaaaaa"}, []string{"bbbbbb"}, nil)
	id, err := NewSchedulers(ws).Add(t.Context(), map[string]any{"name": "x"})
	if err != nil || id != "bbbbbb" {
		t.Fatalf("Add = %q, %v", id, err)
	}
	if got := fmt.Sprint(*calls); got != "[map[domain:scheduler service:add service_data:map[name:x]]]" {
		t.Errorf("calls = %s", got)
	}

	ws, _ = fakeSchedulerWS(nil, []string{"bbbbbb"}, []string{"cccccc"})
	if _, err := NewSchedulers(ws).Add(t.Context(), nil); err == nil || !strings.Contains(err.Error(), "several schedules") ||
		!strings.Contains(err.Error(), "tofu import") {
		t.Errorf("concurrent add: %v", err)
	}
	ws, _ = fakeSchedulerWS(nil, nil, nil)
	if _, err := NewSchedulers(ws).Add(t.Context(), nil); err == nil || !strings.Contains(err.Error(), "no new schedule") {
		t.Errorf("nothing added: %v", err)
	}
}

func TestSchedulersEditAndEnable(t *testing.T) {
	ws, calls := fakeSchedulerWS(nil, nil, nil)
	s := NewSchedulers(ws)
	fields := map[string]any{"weekdays": []string{"mon"}}
	if err := s.Edit(t.Context(), "switch.schedule_x", fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["entity_id"]; ok {
		t.Error("Edit changed its argument")
	}
	if err := s.SetEnabled(t.Context(), "switch.schedule_x", false); err != nil {
		t.Fatal(err)
	}
	want := "[map[domain:scheduler service:edit service_data:map[entity_id:switch.schedule_x weekdays:[mon]]] " +
		"map[domain:switch service:turn_off service_data:map[entity_id:switch.schedule_x]]]"
	if got := fmt.Sprint(*calls); got != want {
		t.Errorf("calls = %s\nwant    %s", got, want)
	}
}

func TestSchedulersGet(t *testing.T) {
	s := NewSchedulers(fakeWS{"scheduler/item": fixed(`null`)})
	if _, ok, err := s.Get(t.Context(), "x"); ok || err != nil {
		t.Errorf("missing: %v, %v", ok, err)
	}
	s = NewSchedulers(fakeWS{"scheduler/item": fixed(`{"schedule_id":"x","name":"N","entity_id":"switch.schedule_n"}`)})
	if got, ok, err := s.Get(t.Context(), "x"); !ok || err != nil || got.EntityID != "switch.schedule_n" || *got.Name != "N" {
		t.Errorf("Get = %+v, %v, %v", got, ok, err)
	}
}

func TestSchedulerMissing(t *testing.T) {
	// Home Assistant without the integration knows neither the commands nor the services.
	s := NewSchedulers(fakeWS{"call_service": func(p map[string]any) (any, error) {
		return nil, &WSError{Code: "not_found", Message: fmt.Sprintf("Service %s.%s not found.", p["domain"], p["service"])}
	}})
	if _, err := s.List(t.Context()); !errors.Is(err, ErrSchedulerMissing) {
		t.Errorf("List: %v", err)
	}
	if err := s.Remove(t.Context(), "switch.schedule_x"); !errors.Is(err, ErrSchedulerMissing) {
		t.Errorf("Remove: %v", err)
	}
	// A missing switch is another error.
	if err := s.SetEnabled(t.Context(), "switch.schedule_x", true); errors.Is(err, ErrSchedulerMissing) {
		t.Errorf("SetEnabled: %v", err)
	}
	s = NewSchedulers(fakeWS{"call_service": func(map[string]any) (any, error) {
		return nil, &WSError{Code: "invalid_format", Message: "Entity not found: switch.nope"}
	}})
	if err := s.Edit(t.Context(), "switch.nope", nil); errors.Is(err, ErrSchedulerMissing) || err == nil {
		t.Errorf("Edit: %v", err)
	}
}

func TestSchedulersAddRetriesList(t *testing.T) {
	schedulerListRetryDelay = time.Millisecond
	t.Cleanup(func() { schedulerListRetryDelay = time.Second })

	// The list fails once after the add, then works: the ID is still found.
	ws, _ := fakeSchedulerWS(nil, []string{"bbbbbb"}, nil)
	list, lists := ws["scheduler"], 0
	ws["scheduler"] = func(p map[string]any) (any, error) {
		if lists++; lists == 2 {
			return nil, ErrConnectionLost
		}
		return list(p)
	}
	if id, err := NewSchedulers(ws).Add(t.Context(), map[string]any{"name": "x"}); err != nil || id != "bbbbbb" {
		t.Errorf("Add = %q, %v", id, err)
	}

	// The list keeps failing: the error says the schedule exists and how to import it.
	lists = 0
	ws["scheduler"] = func(p map[string]any) (any, error) {
		if lists++; lists > 1 {
			return nil, ErrConnectionLost
		}
		return list(p)
	}
	_, err := NewSchedulers(ws).Add(t.Context(), map[string]any{"name": "Office"})
	if !errors.Is(err, ErrConnectionLost) || !strings.Contains(err.Error(), `created the schedule "Office"`) ||
		!strings.Contains(err.Error(), "tofu import") {
		t.Errorf("Add: %v", err)
	}
}

func TestSchedulersAddCallFails(t *testing.T) {
	failing := func(created bool, callErr error, listFails bool) fakeWS {
		ws, _ := fakeSchedulerWS([]string{"aaaaaa"}, nil, nil)
		list, lists := ws["scheduler"], 0
		ids := []string{"aaaaaa"}
		ws["scheduler"] = func(p map[string]any) (any, error) {
			if lists++; lists > 1 && listFails {
				return nil, ErrConnectionLost
			}
			if _, err := list(p); err != nil {
				return nil, err
			}
			all := make([]map[string]any, len(ids))
			for i, id := range ids {
				all[i] = map[string]any{"schedule_id": id}
			}
			return all, nil
		}
		ws["call_service"] = func(map[string]any) (any, error) {
			if created {
				ids = append(ids, "bbbbbb")
			}
			return nil, callErr
		}
		return ws
	}
	add := func(ws fakeWS) (string, error) {
		return NewSchedulers(ws).Add(t.Context(), map[string]any{"name": "Office"})
	}

	// Lost while in flight, but created: the new ID is found.
	if id, err := add(failing(true, ErrConnectionLost, false)); err != nil || id != "bbbbbb" {
		t.Errorf("created: %q, %v", id, err)
	}
	// Lost while in flight, not created: the original error, without a hint.
	if _, err := add(failing(false, ErrConnectionLost, false)); !errors.Is(err, ErrConnectionLost) ||
		strings.Contains(err.Error(), "tofu import") {
		t.Errorf("not created: %v", err)
	}
	// Rejected by Home Assistant: nothing was created, no second list.
	if _, err := add(failing(false, &WSError{Code: "invalid_format", Message: "bad"}, true)); err == nil ||
		strings.Contains(err.Error(), "tofu import") {
		t.Errorf("rejected: %v", err)
	}
	// Timed out, and the list fails too: it may exist.
	_, err := add(failing(true, context.DeadlineExceeded, true))
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), `may have created the schedule "Office"`) {
		t.Errorf("unknown: %v", err)
	}
}
