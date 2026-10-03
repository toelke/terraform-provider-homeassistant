package client

import "context"

// InstanceConfig is the instance configuration from `GET /api/config`.
type InstanceConfig struct {
	Version      string     `json:"version"`
	LocationName string     `json:"location_name"`
	Latitude     float64    `json:"latitude"`
	Longitude    float64    `json:"longitude"`
	Elevation    float64    `json:"elevation"`
	TimeZone     string     `json:"time_zone"`
	UnitSystem   UnitSystem `json:"unit_system"`
	Components   []string   `json:"components"`
}

// UnitSystem holds the units the instance displays values in.
type UnitSystem struct {
	Temperature string `json:"temperature"`
	Length      string `json:"length"`
	Mass        string `json:"mass"`
	Volume      string `json:"volume"`
}

// Config reads the instance configuration.
func (c *RESTClient) Config(ctx context.Context) (*InstanceConfig, error) {
	var cfg InstanceConfig
	if err := c.Get(ctx, "config", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
