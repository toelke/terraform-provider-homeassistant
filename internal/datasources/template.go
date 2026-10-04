package datasources

import (
	"context"
	"errors"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

var _ datasource.DataSourceWithConfigure = (*templateDataSource)(nil)

// templateRenderer is the part of the client the template data source uses.
type templateRenderer interface {
	RenderTemplate(ctx context.Context, template string) (string, error)
}

// NewTemplate returns the `homeassistant_template` data source.
func NewTemplate() datasource.DataSource {
	return &templateDataSource{}
}

type templateDataSource struct {
	client templateRenderer
}

type templateModel struct {
	Template types.String `tfsdk:"template"`
	Result   types.String `tfsdk:"result"`
}

func (d *templateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template"
}

func (d *templateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Renders a Jinja template in Home Assistant (`POST /api/template`). To get " +
			"structured data, render it with `| tojson` and decode `result` with `jsondecode()`.",
		Attributes: map[string]schema.Attribute{
			"template": schema.StringAttribute{
				Description: "The template, e.g. `{{ areas() | tojson }}`.",
				Required:    true,
			},
			"result": schema.StringAttribute{
				Description: "The rendered text.",
				Computed:    true,
			},
		},
	}
}

func (d *templateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := restClient(req, resp); c != nil {
		d.client = c
	}
}

func (d *templateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg templateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.RenderTemplate(ctx, cfg.Template.ValueString())
	if err != nil {
		if isRenderError(err) {
			resp.Diagnostics.AddAttributeError(path.Root("template"), "Rendering template failed", err.Error())
			return
		}
		resp.Diagnostics.AddError("Rendering template", client.ErrorDetail(err))
		return
	}

	cfg.Result = types.StringValue(result)
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg)...)
}

// isRenderError reports whether err says the template itself is broken: HA answers HTTP 400, or
// HTTP 200 with an error text.
func isRenderError(err error) bool {
	var tmplErr *client.TemplateError
	var httpErr *client.HTTPError
	return errors.As(err, &tmplErr) || errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusBadRequest
}
