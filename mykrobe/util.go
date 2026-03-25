package mykrobe

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"strings"
)

func GetParams(url string) map[string]string {
	params := map[string]string{}
	parts := strings.Split(url, "?")
	if len(parts) < 2 {
		return params
	}
	pstr := strings.Split(parts[1], " ")[0]
	for _, p := range strings.Split(pstr, "&") {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}
	return params
}

func Unique[T comparable](items []T) []T {
	seen := map[T]struct{}{}
	out := make([]T, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func Flatten[T any](items [][]T) []T {
	var out []T
	for _, xs := range items {
		out = append(out, xs...)
	}
	return out
}

func LoadJSON(path string, dst any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	buf := make([]byte, 2)
	if _, err := f.Read(buf); err == nil {
		_, _ = f.Seek(0, 0)
		if buf[0] == 0x1f && buf[1] == 0x8b {
			gr, err := gzip.NewReader(f)
			if err != nil {
				return err
			}
			defer gr.Close()
			r = gr
		}
	}
	return json.NewDecoder(r).Decode(dst)
}
