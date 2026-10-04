package resources

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Validators shared by the resources on REST config editors: automation, script, and scene.

var configItemIDPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

// configItemIDValidator checks a user-chosen ID (ADR-0008) at plan time.
type configItemIDValidator struct{}

var _ validator.String = configItemIDValidator{}

func (configItemIDValidator) Description(context.Context) string {
	return "must consist of lowercase letters, digits, and underscores"
}

func (v configItemIDValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v configItemIDValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if s := req.ConfigValue.ValueString(); !configItemIDPattern.MatchString(s) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Attribute Value",
			fmt.Sprintf("Attribute %s %s, got: %q", req.Path, v.Description(ctx), s))
	}
}

// configItemConfigValidator checks at plan time that a `config` is an object without an `id`
// key: the provider owns the ID and sends it from the `id` attribute.
type configItemConfigValidator struct{}

var _ validator.Dynamic = configItemConfigValidator{}

func (configItemConfigValidator) Description(context.Context) string {
	return "must be an object without an `id` key"
}

func (v configItemConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (configItemConfigValidator) ValidateDynamic(_ context.Context, req validator.DynamicRequest, resp *validator.DynamicResponse) {
	cv := req.ConfigValue
	if cv.IsNull() || cv.IsUnknown() || cv.IsUnderlyingValueNull() || cv.IsUnderlyingValueUnknown() {
		return
	}
	var keys map[string]attr.Value
	switch u := cv.UnderlyingValue().(type) {
	case basetypes.ObjectValue:
		keys = u.Attributes()
	case basetypes.MapValue:
		keys = u.Elements()
	default:
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid config",
			fmt.Sprintf("config must be an object, got %s.", u.Type(context.Background())))
		return
	}
	if _, ok := keys["id"]; ok {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid config",
			"config must not contain an `id` key. Set the top-level `id` attribute instead; the "+
				"provider sends it to Home Assistant.")
	}
}
