package mykrobe

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
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
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	if err := enc.Encode(NormalizeForPythonJSON(v)); err != nil {
		return err
	}
	_, err := w.Write(unescapeHTMLJSON(buf.Bytes()))
	return err
}

func marshalJSONLikePython(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := WriteJSONLikePython(&buf, v, ""); err != nil {
		return nil, err
	}
	return unescapeHTMLJSON([]byte(strings.TrimSuffix(buf.String(), "\n"))), nil
}

func unescapeHTMLJSON(data []byte) []byte {
	out := string(data)
	out = strings.ReplaceAll(out, `\u0026`, "&")
	out = strings.ReplaceAll(out, `\u003c`, "<")
	out = strings.ReplaceAll(out, `\u003e`, ">")
	return []byte(out)
}
