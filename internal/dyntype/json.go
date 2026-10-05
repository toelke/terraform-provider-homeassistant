package dyntype

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// numberPrecision matches the precision Terraform uses for numbers.
const numberPrecision = 512

// JSON encodes the value as JSON, unchanged apart from key order, for sending to HA (ADR-0005).
// It fails if the value is null or not wholly known.
func (v Value) JSON() ([]byte, error) {
	if !isWhollyKnown(v.DynamicValue) {
		return nil, errors.New("dynamic config is null or not known")
	}
	g, err := v.goValue()
	if err != nil {
		return nil, err
	}
	return json.Marshal(g)
}

// FromJSON decodes JSON read from HA into a dynamic config value. Objects become object values,
// arrays become tuple values, and JSON null becomes a dynamic null.
func FromJSON(data []byte) (Value, error) {
	g, err := decodeJSON(data)
	if err != nil {
		return Value{}, err
	}
	av, err := fromGo(g)
	if err != nil {
		return Value{}, err
	}
	return NewValue(av), nil
}

// SemanticallyEqualJSON reports whether two JSON documents are equal after normalisation.
func SemanticallyEqualJSON(a, b []byte) (bool, error) {
	ga, err := decodeJSON(a)
	if err != nil {
		return false, err
	}
	gb, err := decodeJSON(b)
	if err != nil {
		return false, err
	}
	return semanticallyEqual(ga, gb)
}

func decodeJSON(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var g any
	if err := dec.Decode(&g); err != nil {
		return nil, fmt.Errorf("decoding JSON: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("decoding JSON: unexpected data after top-level value")
	}
	return g, nil
}

func semanticallyEqual(a, b any) (bool, error) {
	ja, err := canonicalJSON(a)
	if err != nil {
		return false, err
	}
	jb, err := canonicalJSON(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(ja, jb), nil
}

// Hash returns the hex SHA-256 of a JSON document's normalised form, so two documents have the
// same hash exactly when they are semantically equal. It is the stored baseline (ADR-0023).
func Hash(data []byte) (string, error) {
	g, err := decodeJSON(data)
	if err != nil {
		return "", err
	}
	j, err := canonicalJSON(g)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(j)
	return hex.EncodeToString(sum[:]), nil
}

// canonicalJSON encodes a decoded JSON tree in normalised form.
func canonicalJSON(g any) ([]byte, error) {
	n, err := normalize(g)
	if err != nil {
		return nil, err
	}
	return json.Marshal(n)
}

// normalize returns a copy of a decoded JSON tree in canonical form. Together with
// json.Marshal, which sorts object keys, it defines semantic equality. Rules:
//
//   - Numbers are written canonically: integers exactly (5 and 5.0 are both "5"), other numbers
//     as the shortest float64 representation, because HA stores them as Python floats.
//   - Object keys whose value is null are dropped: HA drops keys set to null when it stores
//     config, so `{a = null}` and `{}` mean the same thing.
//   - Nulls inside arrays are kept, because their position is significant.
func normalize(g any) (any, error) {
	switch x := g.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			if e == nil {
				continue
			}
			n, err := normalize(e)
			if err != nil {
				return nil, err
			}
			out[k] = n
		}
		return out, nil
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			n, err := normalize(e)
			if err != nil {
				return nil, err
			}
			out[i] = n
		}
		return out, nil
	case json.Number:
		f, _, err := big.ParseFloat(string(x), 10, numberPrecision, big.ToNearestEven)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q: %w", x, err)
		}
		return canonicalNumber(f), nil
	default:
		return g, nil
	}
}

func canonicalNumber(f *big.Float) json.Number {
	if f.IsInt() {
		return json.Number(f.Text('f', 0))
	}
	f64, _ := f.Float64()
	return json.Number(strconv.FormatFloat(f64, 'g', -1, 64))
}

// goValue converts the value into a decoded JSON tree: map[string]any, []any, string, bool,
// json.Number, or nil.
func (v Value) goValue() (any, error) {
	return GoValue(v)
}

func toGo(tv tftypes.Value) (any, error) {
	if !tv.IsKnown() {
		return nil, errors.New("value is not known")
	}
	if tv.IsNull() {
		return nil, nil
	}
	typ := tv.Type()
	switch {
	case typ.Is(tftypes.String):
		var s string
		err := tv.As(&s)
		return s, err
	case typ.Is(tftypes.Bool):
		var b bool
		err := tv.As(&b)
		return b, err
	case typ.Is(tftypes.Number):
		f := new(big.Float)
		if err := tv.As(&f); err != nil {
			return nil, err
		}
		return canonicalNumber(f), nil
	case typ.Is(tftypes.List{}), typ.Is(tftypes.Set{}), typ.Is(tftypes.Tuple{}):
		var elems []tftypes.Value
		if err := tv.As(&elems); err != nil {
			return nil, err
		}
		out := make([]any, len(elems))
		for i, e := range elems {
			g, err := toGo(e)
			if err != nil {
				return nil, err
			}
			out[i] = g
		}
		return out, nil
	case typ.Is(tftypes.Map{}), typ.Is(tftypes.Object{}):
		var attrs map[string]tftypes.Value
		if err := tv.As(&attrs); err != nil {
			return nil, err
		}
		out := make(map[string]any, len(attrs))
		for k, e := range attrs {
			g, err := toGo(e)
			if err != nil {
				return nil, err
			}
			out[k] = g
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported type %s in dynamic config", typ)
	}
}

func fromGo(g any) (attr.Value, error) {
	ctx := context.Background()
	switch x := g.(type) {
	case nil:
		return basetypes.NewDynamicNull(), nil
	case string:
		return basetypes.NewStringValue(x), nil
	case bool:
		return basetypes.NewBoolValue(x), nil
	case json.Number:
		f, _, err := big.ParseFloat(string(x), 10, numberPrecision, big.ToNearestEven)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q: %w", x, err)
		}
		return basetypes.NewNumberValue(f), nil
	case []any:
		types := make([]attr.Type, len(x))
		elems := make([]attr.Value, len(x))
		for i, e := range x {
			av, err := fromGo(e)
			if err != nil {
				return nil, err
			}
			types[i] = av.Type(ctx)
			elems[i] = av
		}
		tv, diags := basetypes.NewTupleValue(types, elems)
		if diags.HasError() {
			return nil, fmt.Errorf("building tuple: %v", diags)
		}
		return tv, nil
	case map[string]any:
		types := make(map[string]attr.Type, len(x))
		attrs := make(map[string]attr.Value, len(x))
		for k, e := range x {
			av, err := fromGo(e)
			if err != nil {
				return nil, err
			}
			types[k] = av.Type(ctx)
			attrs[k] = av
		}
		ov, diags := basetypes.NewObjectValue(types, attrs)
		if diags.HasError() {
			return nil, fmt.Errorf("building object: %v", diags)
		}
		return ov, nil
	default:
		return nil, fmt.Errorf("unsupported JSON value %T", g)
	}
}

// GoValue converts a wholly known framework value of any type into a decoded JSON tree:
// map[string]any, []any, string, bool, json.Number, or nil.
func GoValue(v attr.Value) (any, error) {
	tv, err := v.ToTerraformValue(context.Background())
	if err != nil {
		return nil, err
	}
	return toGo(tv)
}

// FromGoValue converts a decoded JSON tree, as returned by GoValue, into a framework value:
// objects become object values and arrays become tuple values.
func FromGoValue(g any) (attr.Value, error) {
	return fromGo(g)
}
