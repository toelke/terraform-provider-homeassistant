package client

import (
	"context"
	"encoding/json"
)

// EntityState is one entity's state from `GET /api/states/<entity_id>`.
type EntityState struct {
	EntityID string `json:"entity_id"`
	State    string `json:"state"`
	// Attributes is the raw JSON object, so that callers keep its JSON types (ADR-0015).
	Attributes  json.RawMessage `json:"attributes"`
	LastChanged string          `json:"last_changed"`
	LastUpdated string          `json:"last_updated"`
}

// State reads the state of one entity. A missing entity is ErrNotFound.
func (c *RESTClient) State(ctx context.Context, entityID string) (*EntityState, error) {
	var s EntityState
	if err := c.Get(ctx, "states/"+entityID, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// States reads the state of every entity (`GET /api/states`).
func (c *RESTClient) States(ctx context.Context) ([]EntityState, error) {
	var states []EntityState
	if err := c.Get(ctx, "states", &states); err != nil {
		return nil, err
	}
	return states, nil
}
