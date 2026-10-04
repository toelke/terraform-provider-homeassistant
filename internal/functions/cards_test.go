package functions

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/dyntype"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

// dyn returns the JSON document as a dynamic argument.
func dyn(t *testing.T, doc string) attr.Value {
	t.Helper()
	v, err := dyntype.FromJSON([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	return v.DynamicValue
}

// options returns the variadic argument: one dynamic element per JSON document.
func options(t *testing.T, docs ...string) attr.Value {
	t.Helper()
	typs := make([]attr.Type, len(docs))
	elems := make([]attr.Value, len(docs))
	for i, d := range docs {
		typs[i] = types.DynamicType
		elems[i] = dyn(t, d)
	}
	return types.TupleValueMust(typs, elems)
}

func run(t *testing.T, f function.Function, args ...attr.Value) function.RunResponse {
	t.Helper()
	ctx := context.Background()
	resp := function.RunResponse{Result: function.NewResultData(types.DynamicUnknown())}
	f.Run(ctx, function.RunRequest{Arguments: function.NewArgumentsData(args)}, &resp)
	return resp
}

// TestGolden compares each function's result with testdata/<name>.json. Run with -update to
// rewrite the files.
func TestGolden(t *testing.T) {
	tests := []struct {
		name string
		f    function.Function
		args func(t *testing.T) []attr.Value
	}{
		{"tile_card", NewTileCard(), func(t *testing.T) []attr.Value {
			return []attr.Value{types.StringValue("light.kitchen"), options(t)}
		}},
		{"tile_card_options", NewTileCard(), func(t *testing.T) []attr.Value {
			return []attr.Value{
				types.StringValue("light.kitchen"),
				options(t, `{"name":"Kitchen","icon":null,"features":[{"type":"light-brightness"}]}`),
			}
		}},
		{"entities_card", NewEntitiesCard(), func(t *testing.T) []attr.Value {
			return []attr.Value{
				dyn(t, `["light.kitchen",{"entity":"light.hall","name":"Hall"}]`),
				options(t, `{"title":"Lights","show_header_toggle":false}`),
			}
		}},
		{"heading_card", NewHeadingCard(), func(t *testing.T) []attr.Value {
			return []attr.Value{types.StringValue("Quick actions"), options(t)}
		}},
		{"grid_section", NewGridSection(), func(t *testing.T) []attr.Value {
			return []attr.Value{
				dyn(t, `[{"type":"tile","entity":"light.kitchen"},{"type":"button","entity":"script.good_morning"}]`),
				types.StringValue("Kitchen"),
				options(t, `{"column_span":2}`),
			}
		}},
		{"grid_section_no_heading", NewGridSection(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `[{"type":"tile","entity":"lock.front_door"}]`), types.StringValue(""), options(t)}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := run(t, tt.f, tt.args(t)...)
			if resp.Error != nil {
				t.Fatal(resp.Error)
			}
			g, err := dyntype.GoValue(resp.Result.Value())
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.MarshalIndent(g, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, '\n')

			path := filepath.Join("testdata", tt.name+".json")
			if *update {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("result differs from %s:\ngot:\n%s\nwant:\n%s", path, got, want)
			}
		})
	}
}

func TestInvalidArguments(t *testing.T) {
	tests := []struct {
		name     string
		f        function.Function
		args     func(t *testing.T) []attr.Value
		position int64
		message  string
	}{
		{"options set a key that an argument sets", NewTileCard(), func(t *testing.T) []attr.Value {
			return []attr.Value{types.StringValue("light.kitchen"), options(t, `{"entity":"light.hall"}`)}
		}, 1, `options must not set "entity"`},
		{"options set the type", NewHeadingCard(), func(t *testing.T) []attr.Value {
			return []attr.Value{types.StringValue("Lights"), options(t, `{"type":"markdown"}`)}
		}, 1, `options must not set "type"`},
		{"two options objects", NewTileCard(), func(t *testing.T) []attr.Value {
			return []attr.Value{types.StringValue("light.kitchen"), options(t, `{}`, `{}`)}
		}, 2, "at most one options object"},
		{"options not an object", NewTileCard(), func(t *testing.T) []attr.Value {
			return []attr.Value{types.StringValue("light.kitchen"), options(t, `"big"`)}
		}, 1, "options must be an object"},
		{"entities not a list", NewEntitiesCard(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `"light.kitchen"`), options(t)}
		}, 0, "entities must be a list"},
		{"entity row neither string nor object", NewEntitiesCard(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `["light.kitchen",5]`), options(t)}
		}, 0, "entities[1] must be an entity ID or an object"},
		{"cards not a list", NewGridSection(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `{"type":"tile"}`), types.StringValue(""), options(t)}
		}, 0, "cards must be a list of objects"},
		{"card not an object", NewGridSection(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `[{"type":"tile"},"light.kitchen"]`), types.StringValue(""), options(t)}
		}, 0, "cards[1] must be an object"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := run(t, tt.f, tt.args(t)...)
			if resp.Error == nil {
				t.Fatalf("no error; result %v", resp.Result.Value())
			}
			if resp.Error.FunctionArgument == nil || *resp.Error.FunctionArgument != tt.position {
				t.Errorf("error is at argument %v, want %d: %s", resp.Error.FunctionArgument, tt.position, resp.Error)
			}
			if !strings.Contains(resp.Error.Text, tt.message) {
				t.Errorf("error %q does not contain %q", resp.Error.Text, tt.message)
			}
		})
	}
}
