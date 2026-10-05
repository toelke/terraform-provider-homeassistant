package resources

import (
	"encoding/json"
	"testing"

	"github.com/toelke/terraform-provider-homeassistant/internal/dyntype"
)

func TestRefreshedConfig(t *testing.T) {
	// The user's config with old keys, and what HA stores for it.
	const written = `{"alias":"Legacy","trigger":[{"platform":"state","entity_id":"sun.sun"}],
		"action":[{"service":"light.turn_off"}]}`
	const stored = `{"alias":"Legacy","triggers":[{"platform":"state","entity_id":"sun.sun"}],
		"actions":[{"action":"light.turn_off"}]}`
	const edited = `{"alias":"Edited in the UI","triggers":[],"actions":[]}`

	prior, err := dyntype.FromJSON([]byte(written))
	if err != nil {
		t.Fatal(err)
	}
	hash, err := dyntype.Hash([]byte(stored))
	if err != nil {
		t.Fatal(err)
	}
	baseline, _ := json.Marshal(hash)

	cases := map[string]struct {
		prior    dyntype.Value
		read     string
		baseline []byte
		want     string
	}{
		"equal to the baseline keeps prior":   {prior, stored, baseline, written},
		"baseline matches any key order":      {prior, `{"actions":[{"action":"light.turn_off"}],"alias":"Legacy","triggers":[{"entity_id":"sun.sun","platform":"state"}]}`, baseline, written},
		"changed outside Tofu is read":        {prior, edited, baseline, edited},
		"no baseline returns the config read": {prior, stored, nil, stored},
		"null prior ignores the baseline":     {dyntype.NewNull(), stored, baseline, stored},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := refreshedConfig(tc.prior, json.RawMessage(tc.read), tc.baseline)
			if err != nil {
				t.Fatal(err)
			}
			gotJSON, err := got.JSON()
			if err != nil {
				t.Fatal(err)
			}
			eq, err := dyntype.SemanticallyEqualJSON(gotJSON, []byte(tc.want))
			if err != nil {
				t.Fatal(err)
			}
			if !eq {
				t.Errorf("refreshedConfig = %s, want %s", gotJSON, tc.want)
			}
		})
	}
}

// Without a baseline, the config read back is what the framework compares with the prior value,
// so HA's renamed keys are a difference, as in v0.1.0.
func TestRefreshedConfigWithoutBaselineIsSemanticEquality(t *testing.T) {
	prior, err := dyntype.FromJSON([]byte(`{"alias":"x","action":[{"service":"light.turn_off"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	got, err := refreshedConfig(prior, json.RawMessage(`{"alias":"x","actions":[{"action":"light.turn_off"}]}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if eq, diags := prior.DynamicSemanticEquals(t.Context(), got); diags.HasError() || eq {
		t.Errorf("semantically equal = %v (%v), want a difference", eq, diags)
	}
	got, err = refreshedConfig(prior, json.RawMessage(`{"action":[{"service":"light.turn_off"}],"alias":"x"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if eq, diags := prior.DynamicSemanticEquals(t.Context(), got); diags.HasError() || !eq {
		t.Errorf("semantically equal = %v (%v), want equal", eq, diags)
	}
}

func TestRefreshedConfigRejectsInvalidBaseline(t *testing.T) {
	prior, err := dyntype.FromJSON([]byte(`{"alias":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := refreshedConfig(prior, json.RawMessage(`{"alias":"x"}`), []byte(`{}`)); err == nil {
		t.Error("want an error for a baseline that is not a JSON string")
	}
}
