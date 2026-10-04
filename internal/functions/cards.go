package functions

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

// NewTileCard returns the tile_card function.
func NewTileCard() function.Function {
	return builder{
		name:    "tile_card",
		summary: "Builds a tile card",
		description: "Returns a Lovelace `tile` card for one entity, for use in a dashboard " +
			"`config`. Further card options, e.g. `name` or `features`, go in `options`.",
		params: []function.Parameter{
			function.StringParameter{Name: "entity", MarkdownDescription: "Entity ID, e.g. `light.kitchen`."},
		},
		build: func(args []any, _ map[string]any) (map[string]any, *function.FuncError) {
			return map[string]any{"type": "tile", "entity": args[0]}, nil
		},
	}
}

// NewEntitiesCard returns the entities_card function.
func NewEntitiesCard() function.Function {
	return builder{
		name:    "entities_card",
		summary: "Builds an entities card",
		description: "Returns a Lovelace `entities` card, for use in a dashboard `config`. " +
			"Further card options, e.g. `title`, go in `options`.",
		params: []function.Parameter{
			function.DynamicParameter{
				Name: "entities",
				MarkdownDescription: "List of rows: each an entity ID, or an entity row object such as " +
					"`{ entity = \"light.kitchen\", name = \"Ceiling\" }`.",
			},
		},
		build: func(args []any, _ map[string]any) (map[string]any, *function.FuncError) {
			rows, ok := args[0].([]any)
			if !ok {
				return nil, function.NewArgumentFuncError(0, "entities must be a list")
			}
			for i, r := range rows {
				switch r.(type) {
				case string, map[string]any:
				default:
					return nil, function.NewArgumentFuncError(0,
						fmt.Sprintf("entities[%d] must be an entity ID or an object", i))
				}
			}
			return map[string]any{"type": "entities", "entities": rows}, nil
		},
	}
}

// NewHeadingCard returns the heading_card function.
func NewHeadingCard() function.Function {
	return builder{
		name:    "heading_card",
		summary: "Builds a heading card",
		description: "Returns a Lovelace `heading` card, for use in a dashboard `config`. " +
			"Further card options, e.g. `icon` or `heading_style`, go in `options`.",
		params: []function.Parameter{
			function.StringParameter{Name: "heading", MarkdownDescription: "Heading text."},
		},
		build: func(args []any, _ map[string]any) (map[string]any, *function.FuncError) {
			return map[string]any{"type": "heading", "heading": args[0]}, nil
		},
	}
}

// NewGridSection returns the grid_section function.
func NewGridSection() function.Function {
	return builder{
		name:    "grid_section",
		summary: "Builds a grid section",
		description: "Returns a `grid` section of a `sections` view, for use in a dashboard " +
			"`config`. A `heading` in `options` becomes a heading card in front of `cards`. Further " +
			"section options, e.g. `column_span`, go in `options` too.",
		params: []function.Parameter{
			function.DynamicParameter{Name: "cards", MarkdownDescription: "List of card objects."},
		},
		build: func(args []any, options map[string]any) (map[string]any, *function.FuncError) {
			cards, ok := args[0].([]any)
			if !ok {
				return nil, function.NewArgumentFuncError(0, "cards must be a list of objects")
			}
			for i, c := range cards {
				if _, ok := c.(map[string]any); !ok {
					return nil, function.NewArgumentFuncError(0, fmt.Sprintf("cards[%d] must be an object", i))
				}
			}
			if h, set := options["heading"]; set {
				heading, ok := h.(string)
				if !ok {
					return nil, function.NewArgumentFuncError(1, "options.heading must be a string")
				}
				delete(options, "heading")
				cards = append([]any{map[string]any{"type": "heading", "heading": heading}}, cards...)
			}
			return map[string]any{"type": "grid", "cards": cards}, nil
		},
	}
}
