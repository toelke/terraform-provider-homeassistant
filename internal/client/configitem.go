package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// entityPollInterval is how often FindEntity polls `GET /api/states`.
const entityPollInterval = 250 * time.Millisecond

// ConfigItems manages the items of one REST config editor, `/api/config/<domain>/config/<id>`:
// automations, scripts, or scenes. Each item has a user-chosen ID (ADR-0008) and a config object
// (ADR-0005).
type ConfigItems struct {
	rest   *RESTClient
	domain string
	// idInBody is true for domains that store the ID inside the item (automation, scene), as
	// opposed to as its key (script).
	idInBody bool
}

// NewAutomations returns the config editor of automations.
func NewAutomations(rest *RESTClient) ConfigItems {
	return ConfigItems{rest: rest, domain: "automation", idInBody: true}
}

// NewScripts returns the config editor of scripts.
func NewScripts(rest *RESTClient) ConfigItems {
	return ConfigItems{rest: rest, domain: "script"}
}

func (c ConfigItems) path(id string) string {
	return "config/" + c.domain + "/config/" + id
}

// Get returns the config of the item as JSON, without its `id` key. A missing item is
// ErrNotFound.
func (c ConfigItems) Get(ctx context.Context, id string) (json.RawMessage, error) {
	var config map[string]json.RawMessage
	if err := c.rest.Get(ctx, c.path(id), &config); err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("%s %q: config is not an object", c.domain, id)
	}
	delete(config, "id")
	return json.Marshal(config)
}

// Save creates or replaces the item. config is a JSON object without an `id` key. HA validates
// it, and a rejected config is an *HTTPError with status 400 and HA's message. HA reloads the
// domain asynchronously afterwards.
func (c ConfigItems) Save(ctx context.Context, id string, config json.RawMessage) error {
	var body map[string]json.RawMessage
	if err := json.Unmarshal(config, &body); err != nil || body == nil {
		return fmt.Errorf("%s %q: config must be a JSON object", c.domain, id)
	}
	if c.idInBody {
		body["id"], _ = json.Marshal(id)
	}
	return c.rest.Do(ctx, http.MethodPost, c.path(id), body, nil)
}

// Delete deletes the item. A missing item is ErrNotFound.
func (c ConfigItems) Delete(ctx context.Context, id string) error {
	return c.rest.Do(ctx, http.MethodDelete, c.path(id), nil, nil)
}

// ErrEntityNotFound means no entity appeared for a config item before the wait ran out.
var ErrEntityNotFound = errors.New("entity not found")

// FindEntity returns the entity ID of the item. For a domain that stores the ID inside the item,
// that is the entity of the domain whose `attributes.id` is id; for one that stores it as the key
// (script), it is `<domain>.<id>`. Because HA reloads asynchronously after Save, it polls `GET /api/states` until the
// entity appears or wait has passed, and then returns ErrEntityNotFound. A wait of zero looks
// once.
func (c ConfigItems) FindEntity(ctx context.Context, id string, wait time.Duration) (string, error) {
	deadline := time.Now().Add(wait)
	for {
		var states []struct {
			EntityID   string `json:"entity_id"`
			Attributes struct {
				ID any `json:"id"`
			} `json:"attributes"`
		}
		if err := c.rest.Get(ctx, "states", &states); err != nil {
			return "", err
		}
		for _, s := range states {
			if c.matches(s.EntityID, s.Attributes.ID, id) {
				return s.EntityID, nil
			}
		}
		if !time.Now().Before(deadline) {
			if !c.idInBody {
				return "", fmt.Errorf("%w: no entity %s.%s after %s", ErrEntityNotFound, c.domain, id, wait)
			}
			return "", fmt.Errorf("%w: no %s entity has attributes.id %q after %s", ErrEntityNotFound, c.domain, id, wait)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(entityPollInterval):
		}
	}
}

// matches reports whether the state with entityID and attribute `id` attrID is the entity of
// the item id.
func (c ConfigItems) matches(entityID string, attrID any, id string) bool {
	if !c.idInBody {
		return entityID == c.domain+"."+id
	}
	domain, _, _ := strings.Cut(entityID, ".")
	return domain == c.domain && attrID == id
}
