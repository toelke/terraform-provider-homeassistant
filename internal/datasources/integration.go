package datasources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

var _ datasource.DataSourceWithConfigure = (*integrationDataSource)(nil)

// NewIntegration returns the `homeassistant_integration` data source.
func NewIntegration() datasource.DataSource {
	return &integrationDataSource{}
}

type integrationDataSource struct {
	entries client.ConfigEntries
}

type integrationModel struct {
	ID         types.String `tfsdk:"id"`
	EntryID    types.String `tfsdk:"entry_id"`
	Domain     types.String `tfsdk:"domain"`
	Title      types.String `tfsdk:"title"`
	State      types.String `tfsdk:"state"`
	DisabledBy types.String `tfsdk:"disabled_by"`
}

func (d *integrationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

func (d *integrationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads one config entry of an integration, e.g. one set up in the UI. Exactly " +
			"one entry must match `domain` and, if set, `title`. The data the entry was created " +
			"with cannot be read.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				Description: "Integration domain, e.g. `met`.",
				Required:    true,
			},
			"title": schema.StringAttribute{
				Description: "Title of the entry, as shown in the UI. Set it to pick one of several " +
					"entries of the domain.",
				Optional: true,
				Computed: true,
			},
			"id": schema.StringAttribute{
				Description: "The entry's `entry_id`.",
				Computed:    true,
			},
			"entry_id": schema.StringAttribute{
				Description: "The entry's `entry_id`, the same as `id`.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "State of the entry, e.g. `loaded` or `setup_error`.",
				Computed:    true,
			},
			"disabled_by": schema.StringAttribute{
				Description: "Who disabled the entry, e.g. `user`; null if it is enabled.",
				Computed:    true,
			},
		},
	}
}

func (d *integrationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.HAClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected *client.HAClient, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	d.entries = client.NewConfigEntries(c.WS, c.REST)
}

func (d *integrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config integrationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := config.Domain.ValueString()
	entries, err := d.entries.List(ctx, domain)
	if err != nil {
		resp.Diagnostics.AddError("Reading config entries", client.ErrorDetail(err))
		return
	}
	entry, err := matchEntry(entries, domain, config.Title.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError("Finding the config entry", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, integrationModel{
		ID:         types.StringValue(entry.EntryID),
		EntryID:    types.StringValue(entry.EntryID),
		Domain:     types.StringValue(entry.Domain),
		Title:      types.StringValue(entry.Title),
		State:      types.StringValue(entry.State),
		DisabledBy: types.StringPointerValue(entry.DisabledBy),
	})...)
}

// matchEntry returns the one entry of domain with the given title, or the one entry of domain if
// title is nil.
func matchEntry(entries []client.ConfigEntry, domain string, title *string) (client.ConfigEntry, error) {
	var matches []client.ConfigEntry
	for _, e := range entries {
		if e.Domain == domain && (title == nil || e.Title == *title) {
			matches = append(matches, e)
		}
	}
	what := fmt.Sprintf("domain %q", domain)
	if title != nil {
		what += fmt.Sprintf(" and title %q", *title)
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return client.ConfigEntry{}, fmt.Errorf("no config entry has %s", what)
	}
	titles := make([]string, len(matches))
	for i, e := range matches {
		titles[i] = fmt.Sprintf("%q (%s)", e.Title, e.EntryID)
	}
	hint := " Set `title` to pick one."
	if title != nil {
		hint = ""
	}
	return client.ConfigEntry{}, fmt.Errorf("%d config entries have %s: %s.%s",
		len(matches), what, strings.Join(titles, ", "), hint)
}
