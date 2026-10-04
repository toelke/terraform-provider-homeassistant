package datasources

import (
	"strings"
	"testing"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

func TestMatchEntry(t *testing.T) {
	entries := []client.ConfigEntry{
		{EntryID: "E1", Domain: "local_calendar", Title: "Chores"},
		{EntryID: "E2", Domain: "local_calendar", Title: "Trash"},
		{EntryID: "E3", Domain: "met", Title: "Home"},
		{EntryID: "E4", Domain: "local_todo", Title: "Shopping"},
		{EntryID: "E5", Domain: "local_todo", Title: "Shopping"},
	}
	str := func(s string) *string { return &s }

	cases := map[string]struct {
		domain  string
		title   *string
		wantID  string
		wantErr string
	}{
		"only entry of domain": {domain: "met", wantID: "E3"},
		"by title":             {domain: "local_calendar", title: str("Trash"), wantID: "E2"},
		"title of other domain": {
			domain: "met", title: str("Trash"), wantErr: `no config entry has domain "met" and title "Trash"`,
		},
		"no entry": {domain: "shelly", wantErr: `no config entry has domain "shelly"`},
		"ambiguous domain": {
			domain:  "local_calendar",
			wantErr: `2 config entries have domain "local_calendar": "Chores" (E1), "Trash" (E2). Set ` + "`title`",
		},
		"ambiguous title": {
			domain: "local_todo", title: str("Shopping"),
			wantErr: `2 config entries have domain "local_todo" and title "Shopping": "Shopping" (E4), "Shopping" (E5).`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := matchEntry(entries, tc.domain, tc.title)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got.EntryID != tc.wantID {
				t.Errorf("matchEntry = %+v, %v; want %s", got, err, tc.wantID)
			}
		})
	}
}
