package mykrobe

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Mykrobe-tools/mykrobe2/mccortex"
)

// MappingAlias records a mapping key and its distinct, sorted drug names.
type MappingAlias struct {
	Name  string
	Drugs []string
}

// MappingAliasWarning reports aliases for a probe with different drug sets.
// These mappings may be intentional; prediction continues to apply every rule.
type MappingAliasWarning struct {
	Probe   string
	Aliases []MappingAlias
}

func (w MappingAliasWarning) String() string {
	parts := make([]string, 0, len(w.Aliases))
	for _, alias := range w.Aliases {
		parts = append(parts, fmt.Sprintf("%q -> %q", alias.Name, alias.Drugs))
	}
	return fmt.Sprintf("probe %q has aliases with different drug sets: %s; prediction applies all matching rules", w.Probe, strings.Join(parts, "; "))
}

func warnMappingAliases(idx *mccortex.RuntimeIndex, mapping map[string][]string, warn func(MappingAliasWarning)) {
	predictor := &TBPredictor{VariantToResistanceDrug: mapping}
	seen := map[string]bool{}
	for i := range idx.KmerLengths {
		probe := idx.ProbeName(i)
		if !isVariantPanel(strings.SplitN(probe, "?", 2)[0]) {
			continue
		}
		name := variantCallName(probe, GetParams(probe))
		matches := map[string][]string{}
		for _, candidate := range predictor.namesForCall(name, Call{Variant: probe}) {
			if key, drugs, ok := predictor.mappingForName(candidate); ok {
				matches[key] = drugs
			}
		}
		if len(matches) < 2 {
			continue
		}
		keys := make([]string, 0, len(matches))
		for key := range matches {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		signature := strings.Join(keys, "\x00")
		if seen[signature] {
			continue
		}
		seen[signature] = true
		aliases := make([]MappingAlias, 0, len(keys))
		different := false
		for _, key := range keys {
			drugs := slices.Clone(matches[key])
			slices.Sort(drugs)
			drugs = slices.Compact(drugs)
			if len(aliases) > 0 && !slices.Equal(aliases[0].Drugs, drugs) {
				different = true
			}
			aliases = append(aliases, MappingAlias{Name: key, Drugs: drugs})
		}
		if different {
			warn(MappingAliasWarning{Probe: probe, Aliases: aliases})
		}
	}
}
