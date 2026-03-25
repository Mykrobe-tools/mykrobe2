package mykrobe

import (
	"reflect"
	"testing"
)

func TestLineageToSelfPlusParents(t *testing.T) {
	if got := LineageToSelfPlusParents("a"); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatal(got)
	}
	if got := LineageToSelfPlusParents("a.1"); !reflect.DeepEqual(got, []string{"a", "a.1"}) {
		t.Fatal(got)
	}
	if got := LineageToSelfPlusParents("a.1.2"); !reflect.DeepEqual(got, []string{"a", "a.1", "a.1.2"}) {
		t.Fatal(got)
	}
}

func TestLineageConstructorMakesTree(t *testing.T) {
	v := map[string]LineageVariant{
		"var1":     {"lineage1", false, ""},
		"var1a":    {"lineage1", false, ""},
		"var1.2":   {"lineage1.2", false, ""},
		"var1.1.1": {"lineage1.1.1", false, ""},
		"var2":     {"lineage2", false, ""},
		"var2.1":   {"lineage2.1", false, ""},
	}
	p := NewLineagePredictor(v)
	if got := childNames(p.TreeRoot); !reflect.DeepEqual(got, []string{"lineage1", "lineage2"}) {
		t.Fatal(got)
	}
	if got := childNames(p.TreeNodes["lineage1"]); !reflect.DeepEqual(got, []string{"lineage1.1", "lineage1.2"}) {
		t.Fatal(got)
	}
}

func TestLineageScoreEachNodeAndPaths(t *testing.T) {
	v := map[string]LineageVariant{
		"var1":     {"lineage1", true, ""},
		"var1a":    {"lineage1", true, ""},
		"var1.1":   {"lineage1.1", false, ""},
		"var1.2":   {"lineage1.2", false, ""},
		"var1.1.1": {"lineage1.1.1", false, ""},
		"var2":     {"lineage2", false, ""},
		"var2.1":   {"lineage2.1", false, ""},
		"var2.2":   {"lineage2.2", false, ""},
	}
	lineageCalls := map[string]map[string]Call{
		"lineage1":     {"var1": {Genotype: []int{0, 0}, Info: map[string]any{"filter": []string{}, "conf": 500}}, "var1a": {Genotype: []int{1, 1}, Info: map[string]any{"filter": []string{}, "conf": 1000}}},
		"lineage1.1":   {"var1.1": {Genotype: []int{1, 1}, Info: map[string]any{"filter": []string{}, "conf": 1000}}},
		"lineage1.1.1": {"var1.1.1": {Genotype: []int{1, 1}, Info: map[string]any{"filter": []string{}, "conf": 100}}},
		"lineage2":     {"var2": {Genotype: []int{1, 1}, Info: map[string]any{"filter": []string{}, "conf": 1}}},
		"lineage2.1":   {"var2": {Genotype: []int{0, 0}, Info: map[string]any{"filter": []string{}, "conf": 100}}},
	}
	p := NewLineagePredictor(v)
	scores := p.ScoreEachLineageNode(lineageCalls)
	if !reflect.DeepEqual(scores, map[string]float64{"lineage1": 500, "lineage1.1": 1000, "lineage1.1.1": 100, "lineage2": 1, "lineage2.1": -100}) {
		t.Fatal(scores)
	}
	paths := p.GetPathsAndScores(lineageCalls)
	if len(paths) != 3 || paths[0].Lineage != "lineage1.1.1" || paths[0].Score != 1600 {
		t.Fatal(paths)
	}
}

func TestLineageCallLineageUsingConfScores(t *testing.T) {
	v := map[string]LineageVariant{
		"var1":     {"lineage1", false, ""},
		"var1a":    {"lineage1", false, ""},
		"var1.1":   {"lineage1.1", false, ""},
		"var1.1.1": {"lineage1.1.1", false, ""},
		"var2":     {"lineage2", false, ""},
		"var2.1":   {"lineage2.1", false, ""},
	}
	p := NewLineagePredictor(v)
	if got := p.CallLineageUsingConfScores(map[string]map[string]Call{}); got != nil {
		t.Fatal(got)
	}
}

func TestLineageGenotypeAndGoodPathsAndCall(t *testing.T) {
	v := map[string]LineageVariant{
		"var1":     {"lineage1", false, ""},
		"var1a":    {"lineage1", false, ""},
		"var1.1":   {"lineage1.1", false, ""},
		"var1.1.1": {"lineage1.1.1", false, "l1-1-1"},
		"var2":     {"lineage2", false, ""},
		"var2.1":   {"lineage2.1", false, ""},
	}
	lineageCalls := map[string]map[string]Call{
		"lineage1":     {"var1": {Genotype: []int{0, 0}}, "var1a": {Genotype: []int{1, 1}}},
		"lineage1.1":   {"var1.1": {Genotype: []int{1, 1}}},
		"lineage1.1.1": {"var1.1.1": {Genotype: []int{1, 1}}},
		"lineage2":     {"var2": {Genotype: []int{1, 1}}},
		"lineage2.1":   {"var2.1": {Genotype: []int{1, 1}}},
	}
	p := NewLineagePredictor(v)
	genos := p.GenotypeEachLineageNode(lineageCalls)
	if genos["lineage1"] != 1 || genos["lineage1.1"] != 1 {
		t.Fatal(genos)
	}
	good := p.GetGoodPathsUsingGenotypeCalls(lineageCalls, 0.5)
	if _, ok := good["lineage1.1.1"]; !ok {
		t.Fatal(good)
	}
	call := p.CallLineage(lineageCalls, 0.5)
	gotLineages := call["lineage"].([]string)
	if !reflect.DeepEqual(gotLineages, []string{"l1-1-1", "lineage2.1"}) {
		t.Fatal(gotLineages)
	}
}

func childNames(n *lineageNode) []string {
	out := make([]string, len(n.Children))
	for i, c := range n.Children {
		out[i] = c.Name
	}
	return out
}
