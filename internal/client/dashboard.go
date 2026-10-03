package client

import (
	"context"
	"encoding/json"
	"errors"
)

// Dashboard is the metadata of a Lovelace dashboard, as `lovelace/dashboards/list` returns it.
type Dashboard struct {
	// ID is HA's internal dashboard_id, `slugify(url_path)`.
	ID            string  `json:"id"`
	URLPath       string  `json:"url_path"`
	Mode          string  `json:"mode"`
	Title         string  `json:"title"`
	Icon          *string `json:"icon"`
	ShowInSidebar bool    `json:"show_in_sidebar"`
	RequireAdmin  bool    `json:"require_admin"`
}

// DashboardMeta is the settable metadata of a dashboard, other than its url_path.
type DashboardMeta struct {
	Title         string
	Icon          *string
	ShowInSidebar bool
	RequireAdmin  bool
}

// params returns the metadata as command fields. An unset icon is sent as null, which clears it
// on update.
func (m DashboardMeta) params() map[string]any {
	return map[string]any{
		"title":           m.Title,
		"icon":            m.Icon,
		"show_in_sidebar": m.ShowInSidebar,
		"require_admin":   m.RequireAdmin,
	}
}

// ErrNoDashboardConfig means a dashboard exists but has never had a config saved.
var ErrNoDashboardConfig = errors.New("dashboard has no saved config")

// Dashboards manages storage-mode Lovelace dashboards over the WebSocket `lovelace/*` API.
type Dashboards struct {
	ws WSCommander
}

// NewDashboards returns the dashboard API on ws.
func NewDashboards(ws WSCommander) Dashboards {
	return Dashboards{ws: ws}
}

// Get returns the dashboard with the given url_path. ok is false if there is none.
func (d Dashboards) Get(ctx context.Context, urlPath string) (dash Dashboard, ok bool, err error) {
	var all []Dashboard
	if err := d.ws.Command(ctx, "lovelace/dashboards/list", nil, &all); err != nil {
		return dash, false, err
	}
	for _, it := range all {
		if it.URLPath == urlPath {
			return it, true, nil
		}
	}
	return dash, false, nil
}

// Create creates a storage-mode dashboard and returns it as HA stored it.
func (d Dashboards) Create(ctx context.Context, urlPath string, meta DashboardMeta) (Dashboard, error) {
	params := meta.params()
	// Create rejects a null icon, unlike update.
	if meta.Icon == nil {
		delete(params, "icon")
	}
	params["url_path"] = urlPath
	var dash Dashboard
	err := d.ws.Command(ctx, "lovelace/dashboards/create", params, &dash)
	return dash, err
}

// Update sets the metadata of the dashboard with the given dashboard_id.
func (d Dashboards) Update(ctx context.Context, dashboardID string, meta DashboardMeta) error {
	params := meta.params()
	params["dashboard_id"] = dashboardID
	return d.ws.Command(ctx, "lovelace/dashboards/update", params, nil)
}

// Delete deletes the dashboard with the given dashboard_id, including its config.
func (d Dashboards) Delete(ctx context.Context, dashboardID string) error {
	return d.ws.Command(ctx, "lovelace/dashboards/delete", map[string]any{"dashboard_id": dashboardID}, nil)
}

// Config returns the saved Lovelace config of the dashboard as JSON. It returns
// ErrNoDashboardConfig if none was ever saved.
func (d Dashboards) Config(ctx context.Context, urlPath string) (json.RawMessage, error) {
	var cfg json.RawMessage
	err := d.ws.Command(ctx, "lovelace/config", map[string]any{"url_path": urlPath, "force": true}, &cfg)
	var wsErr *WSError
	if errors.As(err, &wsErr) && wsErr.Code == "config_not_found" {
		return nil, ErrNoDashboardConfig
	}
	return cfg, err
}

// SaveConfig replaces the Lovelace config of the dashboard. config is a JSON object.
func (d Dashboards) SaveConfig(ctx context.Context, urlPath string, config json.RawMessage) error {
	return d.ws.Command(ctx, "lovelace/config/save", map[string]any{"url_path": urlPath, "config": config}, nil)
}
