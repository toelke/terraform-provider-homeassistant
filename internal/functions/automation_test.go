package functions

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestAutomationGolden compares each trigger, condition and action function's result with
// testdata/<name>.json.
func TestAutomationGolden(t *testing.T) {
	tests := []struct {
		name string
		f    function.Function
		args func(t *testing.T) []attr.Value
	}{
		{"state_trigger", NewStateTrigger(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `"binary_sensor.motion"`), options(t)}
		}},
		{"state_trigger_options", NewStateTrigger(), func(t *testing.T) []attr.Value {
			return []attr.Value{
				dyn(t, `["binary_sensor.hall","binary_sensor.porch"]`),
				options(t, `{"to":"on","from":null,"for":{"minutes":5}}`),
			}
		}},
		{"numeric_state_trigger", NewNumericStateTrigger(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `"sensor.temperature"`), options(t, `{"above":25,"for":"00:10:00"}`)}
		}},
		{"time_trigger", NewTimeTrigger(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `"07:30:00"`), options(t)}
		}},
		{"time_trigger_list", NewTimeTrigger(), func(t *testing.T) []attr.Value {
			return []attr.Value{
				dyn(t, `["input_datetime.wake_up",{"entity_id":"sensor.phone_next_alarm","offset":"-00:10:00"}]`),
				options(t, `{"weekday":["mon","tue"]}`),
			}
		}},
		{"state_condition", NewStateCondition(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `"person.alice"`), dyn(t, `"home"`), options(t)}
		}},
		{"state_condition_options", NewStateCondition(), func(t *testing.T) []attr.Value {
			return []attr.Value{
				dyn(t, `"alarm_control_panel.home"`),
				dyn(t, `["armed_away","armed_night"]`),
				options(t, `{"for":"00:01:00"}`),
			}
		}},
		{"time_condition", NewTimeCondition(), func(t *testing.T) []attr.Value {
			return []attr.Value{options(t, `{"after":"22:00:00","before":"06:00:00","weekday":null}`)}
		}},
		{"action", NewAction(), func(t *testing.T) []attr.Value {
			return []attr.Value{types.StringValue("homeassistant.reload_all"), options(t)}
		}},
		{"action_options", NewAction(), func(t *testing.T) []attr.Value {
			return []attr.Value{
				types.StringValue("light.turn_on"),
				options(t, `{"target":{"area_id":"kitchen"},"data":{"brightness_pct":80}}`),
			}
		}},
		{"delay", NewDelay(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `"00:00:30"`), options(t)}
		}},
		{"delay_object", NewDelay(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `{"minutes":5}`), options(t, `{"enabled":false}`)}
		}},
		{"delay_seconds", NewDelay(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `1.5`), options(t)}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { checkGolden(t, tt.name, run(t, tt.f, tt.args(t)...)) })
	}
}

func TestAutomationInvalidArguments(t *testing.T) {
	tests := []struct {
		name     string
		f        function.Function
		args     func(t *testing.T) []attr.Value
		position int64
		message  string
	}{
		{"entity_id a number", NewStateTrigger(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `5`), options(t)}
		}, 0, "entity_id must be a string or a list of strings"},
		{"entity_id an empty list", NewStateTrigger(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `[]`), options(t)}
		}, 0, "entity_id must not be an empty list"},
		{"entity_id list with an object", NewNumericStateTrigger(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `["sensor.a",{}]`), options(t, `{"above":1}`)}
		}, 0, "entity_id[1] must be a string"},
		{"options set the trigger", NewStateTrigger(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `"sun.sun"`), options(t, `{"trigger":"event"}`)}
		}, 1, `options must not set "trigger"`},
		{"numeric state without a bound", NewNumericStateTrigger(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `"sensor.temperature"`), options(t, `{"above":null}`)}
		}, 1, "options must set at least one of above, below"},
		{"at a number", NewTimeTrigger(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `730`), options(t)}
		}, 0, "at must be a string, an object, or a list of these"},
		{"at list with a number", NewTimeTrigger(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `["07:30:00",8]`), options(t)}
		}, 0, "at[1] must be a string or an object"},
		{"state an object", NewStateCondition(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `"light.kitchen"`), dyn(t, `{"on":true}`), options(t)}
		}, 1, "state must be a string or a list of strings"},
		{"options set the state", NewStateCondition(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `"light.kitchen"`), dyn(t, `"on"`), options(t, `{"state":"off"}`)}
		}, 2, `options must not set "state"`},
		{"time condition without options", NewTimeCondition(), func(t *testing.T) []attr.Value {
			return []attr.Value{options(t)}
		}, 0, "options must set at least one of after, before, weekday"},
		{"options set the action", NewAction(), func(t *testing.T) []attr.Value {
			return []attr.Value{types.StringValue("light.turn_on"), options(t, `{"action":"light.turn_off"}`)}
		}, 1, `options must not set "action"`},
		{"duration a list", NewDelay(), func(t *testing.T) []attr.Value {
			return []attr.Value{dyn(t, `["00:00:30"]`), options(t)}
		}, 0, "duration must be a string, a number, or an object"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { checkInvalid(t, run(t, tt.f, tt.args(t)...), tt.position, tt.message) })
	}
}
