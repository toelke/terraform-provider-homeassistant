package client

import (
	"context"
	"errors"
	"maps"
)

// EntityEntry is an entry of the entity registry.
type EntityEntry struct {
	EntityID     string   `json:"entity_id"`
	Name         *string  `json:"name"`
	Icon         *string  `json:"icon"`
	AreaID       *string  `json:"area_id"`
	Labels       []string `json:"labels"`
	HiddenBy     *string  `json:"hidden_by"`
	DisabledBy   *string  `json:"disabled_by"`
	Platform     string   `json:"platform"`
	DeviceID     *string  `json:"device_id"`
	OriginalName *string  `json:"original_name"`
}

// EntityRegistry is the entity registry (`config/entity_registry/*`). Integrations create and
// remove its entries; the provider only reads and updates them (ADR-0010).
type EntityRegistry struct {
	ws WSCommander
}

// Entities returns the entity registry.
func Entities(ws WSCommander) EntityRegistry {
	return EntityRegistry{ws: ws}
}

// List returns every entry. Only the acceptance tests use it.
func (r EntityRegistry) List(ctx context.Context) ([]EntityEntry, error) {
	var entries []EntityEntry
	err := r.ws.Command(ctx, "config/entity_registry/list", nil, &entries)
	return entries, err
}

// Get returns the entry of entityID. ok is false if there is none.
func (r EntityRegistry) Get(ctx context.Context, entityID string) (entry EntityEntry, ok bool, err error) {
	err = r.ws.Command(ctx, "config/entity_registry/get", map[string]any{"entity_id": entityID}, &entry)
	var wsErr *WSError
	if errors.As(err, &wsErr) && wsErr.Code == "not_found" {
		return entry, false, nil
	}
	return entry, err == nil, err
}

// Update sets fields on the entry of entityID and returns the result.
func (r EntityRegistry) Update(ctx context.Context, entityID string, fields map[string]any) (EntityEntry, error) {
	params := make(map[string]any, len(fields)+1)
	maps.Copy(params, fields)
	params["entity_id"] = entityID
	var result struct {
		EntityEntry EntityEntry `json:"entity_entry"`
	}
	err := r.ws.Command(ctx, "config/entity_registry/update", params, &result)
	return result.EntityEntry, err
}
