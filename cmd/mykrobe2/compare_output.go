package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type compareOutputOptions struct {
	floatTolerance float64
}

const defaultCompareFloatTolerance = 1e-10

type diffEntry struct {
	Path   string
	Left   any
	Right  any
	Reason string
}

func newCompareOutputCmd() *cobra.Command {
	opts := &compareOutputOptions{}
	cmd := &cobra.Command{
		Use:    "compare_output <left.json> <right.json>",
		Short:  "Compare two mykrobe predict output JSON files",
		Hidden: true,
		Args:   cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompareOutput(opts, cmd.OutOrStdout(), args[0], args[1])
		},
	}
	cmd.Flags().Float64Var(&opts.floatTolerance, "float_tolerance", defaultCompareFloatTolerance, "Absolute float tolerance")
	return cmd
}

func runCompareOutput(opts *compareOutputOptions, out io.Writer, leftPath string, rightPath string) error {
	left, err := loadJSONFile(leftPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", leftPath, err)
	}
	right, err := loadJSONFile(rightPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", rightPath, err)
	}

	leftSamples, ok := left.(map[string]any)
	if !ok {
		return fmt.Errorf("%s does not contain a top-level JSON object", leftPath)
	}
	rightSamples, ok := right.(map[string]any)
	if !ok {
		return fmt.Errorf("%s does not contain a top-level JSON object", rightPath)
	}

	leftHasReport := hasReportAllCalls(leftSamples)
	rightHasReport := hasReportAllCalls(rightSamples)
	if leftHasReport != rightHasReport {
		return fmt.Errorf("report_all_calls mismatch: %s=%t %s=%t", leftPath, leftHasReport, rightPath, rightHasReport)
	}

	summary := compareJSONDocuments(leftSamples, rightSamples, opts.floatTolerance)
	writeCompareSummary(out, summary)
	if summary.Different {
		return fmt.Errorf("outputs differ")
	}
	return nil
}

type compareSummary struct {
	Different bool
	Diffs     []diffEntry
}

func compareJSONDocuments(left map[string]any, right map[string]any, floatTolerance float64) compareSummary {
	var diffs []diffEntry
	diffJSONValue("root", left, right, floatTolerance, &diffs)
	return compareSummary{Different: len(diffs) > 0, Diffs: diffs}
}

func writeCompareSummary(w io.Writer, summary compareSummary) {
	if !summary.Different {
		_, _ = io.WriteString(w, "Outputs match within tolerance.\n")
		return
	}

	_, _ = fmt.Fprintf(w, "Found %d difference(s).\n", len(summary.Diffs))

	sections := []string{
		"root",
		".susceptibility",
		".phylogenetics",
		".variant_calls",
		".sequence_calls",
		".lineage_calls",
	}
	for _, section := range sections {
		writeSectionDiffs(w, summary.Diffs, section)
	}

	var remaining []diffEntry
	for _, diff := range summary.Diffs {
		matched := false
		for _, section := range sections {
			if section == "root" {
				if !strings.Contains(diff.Path, ".susceptibility") &&
					!strings.Contains(diff.Path, ".phylogenetics") &&
					!strings.Contains(diff.Path, ".variant_calls") &&
					!strings.Contains(diff.Path, ".sequence_calls") &&
					!strings.Contains(diff.Path, ".lineage_calls") {
					matched = true
				}
				continue
			}
			if strings.Contains(diff.Path, section) {
				matched = true
				break
			}
		}
		if !matched {
			remaining = append(remaining, diff)
		}
	}
	if len(remaining) > 0 {
		_, _ = io.WriteString(w, "\nOther differences:\n")
		for _, diff := range remaining[:minInt(10, len(remaining))] {
			_, _ = fmt.Fprintf(w, "- %s: %s\n", diff.Path, formatDiff(diff))
		}
	}
}

func writeSectionDiffs(w io.Writer, diffs []diffEntry, section string) {
	var sectionDiffs []diffEntry
	label := section
	if section == "root" {
		label = "top-level"
		for _, diff := range diffs {
			if !strings.Contains(diff.Path, ".susceptibility") &&
				!strings.Contains(diff.Path, ".phylogenetics") &&
				!strings.Contains(diff.Path, ".variant_calls") &&
				!strings.Contains(diff.Path, ".sequence_calls") &&
				!strings.Contains(diff.Path, ".lineage_calls") {
				sectionDiffs = append(sectionDiffs, diff)
			}
		}
	} else {
		for _, diff := range diffs {
			if strings.Contains(diff.Path, section) {
				sectionDiffs = append(sectionDiffs, diff)
			}
		}
	}
	if len(sectionDiffs) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "\n%s differences (%d):\n", label, len(sectionDiffs))
	for _, diff := range sectionDiffs[:minInt(10, len(sectionDiffs))] {
		_, _ = fmt.Fprintf(w, "- %s: %s\n", diff.Path, formatDiff(diff))
	}
}

func formatDiff(diff diffEntry) string {
	switch diff.Reason {
	case "missing_left":
		return fmt.Sprintf("missing from left; right=%s", shortJSON(diff.Right))
	case "missing_right":
		return fmt.Sprintf("missing from right; left=%s", shortJSON(diff.Left))
	case "type":
		return fmt.Sprintf("type mismatch: left=%T right=%T", diff.Left, diff.Right)
	case "length":
		return fmt.Sprintf("length mismatch: left=%s right=%s", shortJSON(diff.Left), shortJSON(diff.Right))
	case "float":
		return fmt.Sprintf("left=%s right=%s", shortJSON(diff.Left), shortJSON(diff.Right))
	default:
		return fmt.Sprintf("left=%s right=%s", shortJSON(diff.Left), shortJSON(diff.Right))
	}
}

func shortJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	s := string(data)
	if len(s) > 140 {
		return s[:137] + "..."
	}
	return s
}

func diffJSONValue(path string, left any, right any, floatTolerance float64, diffs *[]diffEntry) {
	if len(*diffs) >= 200 {
		return
	}
	if left == nil && right == nil {
		return
	}
	if left == nil {
		*diffs = append(*diffs, diffEntry{Path: path, Right: right, Reason: "missing_left"})
		return
	}
	if right == nil {
		*diffs = append(*diffs, diffEntry{Path: path, Left: left, Reason: "missing_right"})
		return
	}

	if lf, ok := asFloat64(left); ok {
		if rf, ok := asFloat64(right); ok {
			if math.Abs(lf-rf) > floatTolerance {
				*diffs = append(*diffs, diffEntry{Path: path, Left: lf, Right: rf, Reason: "float"})
			}
			return
		}
	}

	switch l := left.(type) {
	case map[string]any:
		r, ok := right.(map[string]any)
		if !ok {
			*diffs = append(*diffs, diffEntry{Path: path, Left: left, Right: right, Reason: "type"})
			return
		}
		keysMap := make(map[string]struct{}, len(l)+len(r))
		for k := range l {
			keysMap[k] = struct{}{}
		}
		for k := range r {
			keysMap[k] = struct{}{}
		}
		keys := make([]string, 0, len(keysMap))
		for k := range keysMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			lv, lok := l[k]
			rv, rok := r[k]
			childPath := path + "." + k
			if !lok {
				*diffs = append(*diffs, diffEntry{Path: childPath, Right: rv, Reason: "missing_left"})
				continue
			}
			if !rok {
				*diffs = append(*diffs, diffEntry{Path: childPath, Left: lv, Reason: "missing_right"})
				continue
			}
			diffJSONValue(childPath, lv, rv, floatTolerance, diffs)
			if len(*diffs) >= 200 {
				return
			}
		}
	case []any:
		r, ok := right.([]any)
		if !ok {
			*diffs = append(*diffs, diffEntry{Path: path, Left: left, Right: right, Reason: "type"})
			return
		}
		if len(l) != len(r) {
			*diffs = append(*diffs, diffEntry{Path: path, Left: len(l), Right: len(r), Reason: "length"})
			return
		}
		for i := range l {
			diffJSONValue(fmt.Sprintf("%s[%d]", path, i), l[i], r[i], floatTolerance, diffs)
			if len(*diffs) >= 200 {
				return
			}
		}
	default:
		if !reflect.DeepEqual(left, right) {
			*diffs = append(*diffs, diffEntry{Path: path, Left: left, Right: right, Reason: "value"})
		}
	}
}

func asFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	default:
		return 0, false
	}
}

func hasReportAllCalls(samples map[string]any) bool {
	for _, sampleVariant := range samples {
		sample, ok := sampleVariant.(map[string]any)
		if !ok {
			continue
		}
		if _, ok := sample["variant_calls"]; ok {
			return true
		}
		if _, ok := sample["sequence_calls"]; ok {
			return true
		}
		if _, ok := sample["lineage_calls"]; ok {
			return true
		}
	}
	return false
}

func loadJSONFile(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
