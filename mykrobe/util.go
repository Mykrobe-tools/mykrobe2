package mykrobe

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"regexp"
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

var xVariantPattern = regexp.MustCompile(`^(?P<prefix>.*_)(?P<aa1>[A-Z])(?P<pos1>[0-9]+)(?P<aa2>[A-Z])-(?P<codon1>[ACGT]{3})(?P<pos2>[0-9]+)(?P<codon2>[ACGT]{3})$`)

func XMutationFixedVarName(varName string) (string, bool) {
	m := xVariantPattern.FindStringSubmatch(varName)
	if m == nil {
		return "", false
	}
	g := map[string]string{}
	for i, name := range xVariantPattern.SubexpNames() {
		if i > 0 && name != "" {
			g[name] = m[i]
		}
	}
	if g["aa2"] != "X" {
		return "", false
	}
	codon1AA, ok1 := TranslateCodon(g["codon1"])
	codon1RevAA, ok1r := TranslateCodon(RevcompDNA(g["codon1"]))
	if !ok1 || !ok1r {
		return "", false
	}
	var newAA string
	switch {
	case codon1AA == g["aa1"]:
		x, ok := TranslateCodon(g["codon2"])
		if !ok {
			return "", false
		}
		newAA = x
	case codon1RevAA == g["aa1"]:
		x, ok := TranslateCodon(RevcompDNA(g["codon2"]))
		if !ok {
			return "", false
		}
		newAA = x
	default:
		return "", false
	}
	return g["prefix"] + g["aa1"] + g["pos1"] + newAA + "-" + g["codon1"] + g["pos2"] + g["codon2"], true
}

func FixAminoAcidXVariantKeys(calls map[string]Call) map[string]Call {
	keysToReplace := map[string]string{}
	keysToRemove := map[string]struct{}{}
	for key := range calls {
		newKey, ok := XMutationFixedVarName(key)
		if !ok {
			continue
		}
		if _, exists := keysToReplace[newKey]; exists {
			keysToRemove[key] = struct{}{}
		} else if _, exists := calls[newKey]; exists {
			keysToRemove[key] = struct{}{}
		} else {
			keysToReplace[key] = newKey
		}
	}
	for key := range keysToRemove {
		delete(calls, key)
	}
	for key, newKey := range keysToReplace {
		calls[newKey] = calls[key]
		delete(calls, key)
	}
	return calls
}

func FixAminoAcidXVariantKeysInSusceptibility(susc map[string]map[string]any) map[string]map[string]any {
	for _, drugDict := range susc {
		calledBy, ok := drugDict["called_by"].(map[string]Call)
		if !ok {
			continue
		}
		drugDict["called_by"] = FixAminoAcidXVariantKeys(calledBy)
	}
	return susc
}
