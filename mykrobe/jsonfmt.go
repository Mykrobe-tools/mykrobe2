package mykrobe

import (
	"encoding/json"
	"io"
	"math"
	"reflect"
	"strconv"
)

type pythonFloat float64

func (f pythonFloat) MarshalJSON() ([]byte, error) {
	v := roundJSONFloat(float64(f))
	if math.Abs(v-math.Round(v)) < 1e-12 {
		return []byte(strconv.FormatFloat(v, 'f', 1, 64)), nil
	}
	return []byte(strconv.FormatFloat(v, 'f', -1, 64)), nil
}

func roundJSONFloat(v float64) float64 {
	return math.Round(v*1e11) / 1e11
}

func NormalizeForPythonJSON(v any) any {
	if v == nil {
		return nil
	}
	if _, ok := v.(json.Marshaler); ok {
		return v
	}
	switch x := v.(type) {
	case float64:
		return pythonFloat(x)
	case []float64:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = pythonFloat(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, item := range x {
			out[k] = NormalizeForPythonJSON(item)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = NormalizeForPythonJSON(item)
		}
		return out
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return v
		}
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			out[iter.Key().String()] = NormalizeForPythonJSON(iter.Value().Interface())
		}
		return out
	case reflect.Slice, reflect.Array:
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = NormalizeForPythonJSON(rv.Index(i).Interface())
		}
		return out
	default:
		return v
	}
}

func WriteJSONLikePython(w io.Writer, v any, indent string) error {
	enc := json.NewEncoder(w)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	return enc.Encode(NormalizeForPythonJSON(v))
}
