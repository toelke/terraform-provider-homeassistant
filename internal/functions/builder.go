// Package functions defines the provider's builder functions (ADR-0007): pure functions that
// return plain objects for use inside a dynamic config.
package functions

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/dyntype"
)

// builder is a builder function. Its result is the object returned by build, with the keys of
// the optional trailing options object merged in that build left in place.
type builder struct {
	name        string
	summary     string
	description string
	params      []function.Parameter
	// build turns the arguments, as decoded JSON trees, into the result object, or reports an
	// invalid argument. options holds the options object without its null keys; build deletes
	// the keys it handles itself, and the rest are merged into the result.
	build func(args []any, options map[string]any) (map[string]any, *function.FuncError)
}

var _ function.Function = builder{}

// optionsParameter is the trailing, optional argument of every builder function.
var optionsParameter = function.DynamicParameter{
	Name: "options",
	MarkdownDescription: "At most one object of further keys to set on the result, e.g. " +
		"`{ name = \"Kitchen\" }`. Keys with a null value are left out. It must not set a key " +
		"that another argument already sets.",
}

func (b builder) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = b.name
}

func (b builder) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             b.summary,
		MarkdownDescription: b.description,
		Parameters:          b.params,
		VariadicParameter:   optionsParameter,
		Return:              function.DynamicReturn{},
	}
}

func (b builder) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	values := make([]attr.Value, len(b.params)+1)
	targets := make([]any, len(values))
	for i := range values {
		targets[i] = &values[i]
	}
	if resp.Error = req.Arguments.Get(ctx, targets...); resp.Error != nil {
		return
	}

	args := make([]any, len(values))
	for i, v := range values {
		g, err := dyntype.GoValue(v)
		if err != nil {
			resp.Error = function.NewArgumentFuncError(int64(i), err.Error())
			return
		}
		args[i] = g
	}

	options, funcErr := decodeOptions(args[len(b.params)], len(b.params))
	if funcErr != nil {
		resp.Error = funcErr
		return
	}
	result, funcErr := b.build(args[:len(b.params)], options)
	if funcErr != nil {
		resp.Error = funcErr
		return
	}
	if resp.Error = mergeOptions(result, options, len(b.params)); resp.Error != nil {
		return
	}

	av, err := dyntype.FromGoValue(result)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = resp.Result.Set(ctx, types.DynamicValue(av))
}

// decodeOptions returns the options object without its null keys, or an empty map if there is
// none. variadic is the decoded variadic argument list, and position is the position of its
// first element.
func decodeOptions(variadic any, position int) (map[string]any, *function.FuncError) {
	list, _ := variadic.([]any)
	if len(list) > 1 {
		return nil, function.NewArgumentFuncError(int64(position+1), "at most one options object is allowed")
	}
	options := map[string]any{}
	if len(list) == 0 {
		return options, nil
	}
	given, ok := list[0].(map[string]any)
	if !ok {
		return nil, function.NewArgumentFuncError(int64(position), "options must be an object")
	}
	for k, v := range given {
		if v != nil {
			options[k] = v
		}
	}
	return options, nil
}

// mergeOptions sets the keys of options on result. position is the position of the options
// argument.
func mergeOptions(result, options map[string]any, position int) *function.FuncError {
	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if _, set := result[k]; set {
			return function.NewArgumentFuncError(int64(position),
				fmt.Sprintf("options must not set %q: another argument already sets it", k))
		}
		result[k] = options[k]
	}
	return nil
}
