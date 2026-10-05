package functions

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

// entityIDParameter is the entity_id argument of the triggers and conditions.
var entityIDParameter = function.DynamicParameter{
	Name:                "entity_id",
	MarkdownDescription: "Entity ID, e.g. `binary_sensor.motion`, or a list of entity IDs.",
}

// checkStrings reports an error at position unless v is a string or a non-empty list of
// strings.
func checkStrings(v any, position int64, name string) *function.FuncError {
	switch v := v.(type) {
	case string:
		return nil
	case []any:
		if len(v) == 0 {
			return function.NewArgumentFuncError(position, name+" must not be an empty list")
		}
		for i, e := range v {
			if _, ok := e.(string); !ok {
				return function.NewArgumentFuncError(position, fmt.Sprintf("%s[%d] must be a string", name, i))
			}
		}
		return nil
	}
	return function.NewArgumentFuncError(position, name+" must be a string or a list of strings")
}

// requireOneOf reports an error at position unless options sets at least one of keys.
func requireOneOf(options map[string]any, position int64, keys ...string) *function.FuncError {
	for _, k := range keys {
		if _, set := options[k]; set {
			return nil
		}
	}
	return function.NewArgumentFuncError(position, "options must set at least one of "+strings.Join(keys, ", "))
}

// NewStateTrigger returns the state_trigger function.
func NewStateTrigger() function.Function {
	return builder{
		name:    "state_trigger",
		summary: "Builds a state trigger",
		description: "Returns an automation `state` trigger, for use in `triggers`. Further " +
			"trigger options, e.g. `to`, `from` or `for`, go in `options`. Write `for` as a quoted " +
			"key, `{ \"for\" = \"00:05:00\" }`, because HCL reads `{ for` as a `for` expression.",
		params: []function.Parameter{entityIDParameter},
		build: func(args []any, _ map[string]any) (map[string]any, *function.FuncError) {
			if err := checkStrings(args[0], 0, "entity_id"); err != nil {
				return nil, err
			}
			return map[string]any{"trigger": "state", "entity_id": args[0]}, nil
		},
	}
}

// NewNumericStateTrigger returns the numeric_state_trigger function.
func NewNumericStateTrigger() function.Function {
	return builder{
		name:    "numeric_state_trigger",
		summary: "Builds a numeric state trigger",
		description: "Returns an automation `numeric_state` trigger, for use in `triggers`. " +
			"`options` must set `above`, `below` or both; further trigger options, e.g. " +
			"`attribute` or `for`, go there too.",
		params: []function.Parameter{entityIDParameter},
		build: func(args []any, options map[string]any) (map[string]any, *function.FuncError) {
			if err := checkStrings(args[0], 0, "entity_id"); err != nil {
				return nil, err
			}
			if err := requireOneOf(options, 1, "above", "below"); err != nil {
				return nil, err
			}
			return map[string]any{"trigger": "numeric_state", "entity_id": args[0]}, nil
		},
	}
}

// NewTimeTrigger returns the time_trigger function.
func NewTimeTrigger() function.Function {
	return builder{
		name:    "time_trigger",
		summary: "Builds a time trigger",
		description: "Returns an automation `time` trigger, for use in `triggers`. Further " +
			"trigger options, e.g. `weekday`, go in `options`.",
		params: []function.Parameter{
			function.DynamicParameter{
				Name: "at",
				MarkdownDescription: "A time such as `\"07:30:00\"`, an `input_datetime` or timestamp " +
					"sensor entity ID, an object such as `{ entity_id = \"sensor.alarm\", offset = " +
					"\"-00:10:00\" }`, or a list of these.",
			},
		},
		build: func(args []any, _ map[string]any) (map[string]any, *function.FuncError) {
			at := args[0]
			elems, isList := at.([]any)
			if !isList {
				elems = []any{at}
			} else if len(elems) == 0 {
				return nil, function.NewArgumentFuncError(0, "at must not be an empty list")
			}
			for i, e := range elems {
				switch e.(type) {
				case string, map[string]any:
				default:
					if isList {
						return nil, function.NewArgumentFuncError(0, fmt.Sprintf("at[%d] must be a string or an object", i))
					}
					return nil, function.NewArgumentFuncError(0, "at must be a string, an object, or a list of these")
				}
			}
			return map[string]any{"trigger": "time", "at": at}, nil
		},
	}
}

// NewStateCondition returns the state_condition function.
func NewStateCondition() function.Function {
	return builder{
		name:    "state_condition",
		summary: "Builds a state condition",
		description: "Returns a `state` condition, for use in `conditions`. Further condition " +
			"options, e.g. `attribute` or `for`, go in `options`.",
		params: []function.Parameter{
			entityIDParameter,
			function.DynamicParameter{
				Name:                "state",
				MarkdownDescription: "The state to match, e.g. `\"on\"`, or a list of states, any of which matches.",
			},
		},
		build: func(args []any, _ map[string]any) (map[string]any, *function.FuncError) {
			if err := checkStrings(args[0], 0, "entity_id"); err != nil {
				return nil, err
			}
			if err := checkStrings(args[1], 1, "state"); err != nil {
				return nil, err
			}
			return map[string]any{"condition": "state", "entity_id": args[0], "state": args[1]}, nil
		},
	}
}

// NewTimeCondition returns the time_condition function.
func NewTimeCondition() function.Function {
	return builder{
		name:    "time_condition",
		summary: "Builds a time condition",
		description: "Returns a `time` condition, for use in `conditions`. It has no required " +
			"arguments: `options` must set `after`, `before`, `weekday`, or several of them.",
		build: func(_ []any, options map[string]any) (map[string]any, *function.FuncError) {
			if err := requireOneOf(options, 0, "after", "before", "weekday"); err != nil {
				return nil, err
			}
			return map[string]any{"condition": "time"}, nil
		},
	}
}

// NewAction returns the action function.
func NewAction() function.Function {
	return builder{
		name:    "action",
		summary: "Builds an action call",
		description: "Returns a step that calls an action (formerly called a service), for use " +
			"in `actions` or a script `sequence`. `target`, `data` and further step options, e.g. " +
			"`continue_on_error`, go in `options`.",
		params: []function.Parameter{
			function.StringParameter{Name: "name", MarkdownDescription: "Action to call, e.g. `light.turn_on`."},
		},
		build: func(args []any, _ map[string]any) (map[string]any, *function.FuncError) {
			return map[string]any{"action": args[0]}, nil
		},
	}
}

// NewDelay returns the delay function.
func NewDelay() function.Function {
	return builder{
		name:    "delay",
		summary: "Builds a delay step",
		description: "Returns a step that waits, for use in `actions` or a script `sequence`. " +
			"Further step options, e.g. `enabled`, go in `options`.",
		params: []function.Parameter{
			function.DynamicParameter{
				Name: "duration",
				MarkdownDescription: "How long to wait: `\"HH:MM:SS\"`, a number of seconds, or an " +
					"object such as `{ minutes = 5 }`.",
			},
		},
		build: func(args []any, _ map[string]any) (map[string]any, *function.FuncError) {
			switch args[0].(type) {
			case nil, bool, []any:
				return nil, function.NewArgumentFuncError(0, "duration must be a string, a number, or an object")
			}
			return map[string]any{"delay": args[0]}, nil
		},
	}
}
