// Package resources holds the provider's resources.
package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// generatedID creates objects whose ID HA derives from their name (ADR-0009): floors, areas,
// labels, and input helpers.
type generatedID struct {
	// kind names the object in messages, e.g. "label".
	kind string
	// typeName is the resource type, e.g. "homeassistant_label", for the import hint.
	typeName string
	// create creates the object under name, with all its other fields, and returns the ID HA
	// assigned.
	create func(ctx context.Context, name string) (string, error)
	// rename sets the name of the object with the given ID.
	rename func(ctx context.Context, id, name string) error
	// delete deletes the object with the given ID.
	delete func(ctx context.Context, id string) error
}

// Create creates the object and returns its ID. With id empty, HA derives the ID from name.
// Otherwise id must be a slug; the object is created with id as its name and then renamed to
// name (create-then-rename). If HA assigned a different ID than expected, which it does when the
// expected one is taken, the object is deleted again and Create fails.
func (g generatedID) Create(ctx context.Context, name, id string) (string, error) {
	createName, want := name, client.Slugify(name)
	if id != "" {
		createName, want = id, id
	}

	got, err := g.create(ctx, createName)
	if err != nil {
		return "", fmt.Errorf("creating %s %q: %w. If %s with this name already exists, import it "+
			"into this resource, rename or remove it in Home Assistant, or choose another name",
			g.kind, createName, err, withArticle(g.kind))
	}

	if got != want {
		if delErr := g.delete(ctx, got); delErr != nil {
			return "", fmt.Errorf("home assistant assigned the ID %q instead of %q, and deleting %s %q "+
				"again failed: %w. Delete it in Home Assistant", got, want, g.kind, got, delErr)
		}
		return "", fmt.Errorf("%s with the ID %q already exists, so Home Assistant assigned %q. "+
			"The new %s was deleted again. Either import the existing %s (`tofu import %s.<name> %s`), "+
			"rename or remove it in Home Assistant, or choose another name or `id`",
			withArticle(g.kind), want, got, g.kind, g.kind, g.typeName, want)
	}

	if id != "" && name != id {
		if err := g.rename(ctx, got, name); err != nil {
			if delErr := g.delete(ctx, got); delErr != nil {
				return "", fmt.Errorf("renaming %s %q to %q: %w; deleting it again also failed: %v. "+
					"Delete it in Home Assistant", g.kind, got, name, err, delErr)
			}
			return "", fmt.Errorf("renaming %s %q to %q: %w. The new %s was deleted again. If %s "+
				"with this name already exists, import it into this resource, rename or remove it in Home "+
				"Assistant, or choose another name", g.kind, got, name, err, g.kind, withArticle(g.kind))
		}
	}
	return got, nil
}

// withArticle prefixes kind with "a" or "an", e.g. "an area".
func withArticle(kind string) string {
	if strings.ContainsAny(kind[:1], "aeiou") {
		return "an " + kind
	}
	return "a " + kind
}

// slugValidator accepts strings that HA's slugify leaves unchanged, i.e. valid generated IDs.
type slugValidator struct{}

var _ validator.String = slugValidator{}

func (slugValidator) Description(context.Context) string {
	return "must be a slug: lowercase letters, digits, and single underscores, not at the start or end"
}

func (v slugValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v slugValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	if client.Slugify(s) != s || s == "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Attribute Value",
			fmt.Sprintf("Attribute %s %s, got: %q", req.Path, v.Description(ctx), s))
	}
}
