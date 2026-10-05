package datasources

import (
	"strings"
	"testing"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

func TestMatchArea(t *testing.T) {
	areas := []client.Area{
		{AreaID: "kitchen", Name: "Kitchen"},
		{AreaID: "hall", Name: "Hall"},
		{AreaID: "hall_2", Name: "Hall"}, // HA forbids this, but the lookup must not pick one
	}
	str := func(s string) *string { return &s }

	cases := map[string]struct {
		id, name *string
		wantID   string
		wantErr  string
	}{
		"by ID":           {id: str("kitchen"), wantID: "kitchen"},
		"by name":         {name: str("Kitchen"), wantID: "kitchen"},
		"name is exact":   {name: str("kitchen"), wantErr: `no area has name "kitchen"`},
		"unknown ID":      {id: str("attic"), wantErr: `no area has ID "attic"`},
		"ID is not name":  {id: str("Kitchen"), wantErr: `no area has ID "Kitchen"`},
		"ambiguous names": {name: str("Hall"), wantErr: `2 areas have name "Hall": hall, hall_2;`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := matchArea(areas, tc.id, tc.name)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got.AreaID != tc.wantID {
				t.Errorf("matchArea = %+v, %v; want %s", got, err, tc.wantID)
			}
		})
	}
}
