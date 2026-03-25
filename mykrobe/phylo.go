package mykrobe

import "sort"

const DefaultThreshold = 30.0

var TaxonThresholds = map[string]float64{
	"Saureus":               30,
	"Sepidermidis":          30,
	"Shaemolyticus":         30,
	"Sother":                15,
	"Coagneg":               30,
	"Staphaureus":           30,
	"Escherichia_coli":      15,
	"Klebsiella_pneumoniae": 15,
	"Ecoli_Shigella":        90,
	"Shigella_sonnei":       90,
	"Salmonella_enterica":   90,
	"Salmonella_Typhi":      90,
}

type HierarchyNode struct {
	PhyloGroup string                    `json:"phylo_group"`
	Children   map[string]*HierarchyNode `json:"children"`
}

type Hierarchy struct {
	Dict map[string]*HierarchyNode
}

func LoadHierarchy(path string) (*Hierarchy, error) {
	d := map[string]*HierarchyNode{}
	if path == "" {
		return &Hierarchy{Dict: d}, nil
	}
	if err := LoadJSON(path, &d); err != nil {
		return nil, err
	}
	return &Hierarchy{Dict: d}, nil
}

func (h *Hierarchy) GetChildren(target string) map[string]*HierarchyNode {
	node := h.GetPhyloGroup(target)
	if node == nil {
		return nil
	}
	return node.Children
}

func (h *Hierarchy) GetPhyloGroup(target string) *HierarchyNode {
	for k, v := range h.Dict {
		if k == target {
			return v
		}
		for k2, v2 := range v.Children {
			if k2 == target {
				return v2
			}
			for k3, v3 := range v2.Children {
				if k3 == target {
					return v3
				}
				for k4, v4 := range v3.Children {
					if k4 == target {
						return v4
					}
				}
			}
		}
	}
	return nil
}

type AMRSpeciesPredictor struct {
	PhyloGroupCovgs map[string]map[string]any
	SubComplexCovgs map[string]map[string]any
	SpeciesCovgs    map[string]map[string]any
	LineageCovgs    map[string]map[string]any
	OutJSON         map[string]any
	Threshold       map[string]float64
	Hierarchy       *Hierarchy
}

func NewAMRSpeciesPredictor(phyloGroupCovgs, subComplexCovgs, speciesCovgs, lineageCovgs map[string]map[string]any, hierarchyJSONPath string) (*AMRSpeciesPredictor, error) {
	h, err := LoadHierarchy(hierarchyJSONPath)
	if err != nil {
		return nil, err
	}
	return &AMRSpeciesPredictor{
		PhyloGroupCovgs: phyloGroupCovgs,
		SubComplexCovgs: subComplexCovgs,
		SpeciesCovgs:    speciesCovgs,
		LineageCovgs:    lineageCovgs,
		OutJSON:         map[string]any{},
		Threshold:       TaxonThresholds,
		Hierarchy:       h,
	}, nil
}

func (p *AMRSpeciesPredictor) IsMTBCPresent() bool {
	phylo := p.OutJSON["phylogenetics"].(map[string]any)
	return hasKey(phylo["phylo_group"].(map[string]map[string]any), "Mycobacterium_tuberculosis_complex")
}

func (p *AMRSpeciesPredictor) IsNTMPresent() bool {
	phylo := p.OutJSON["phylogenetics"].(map[string]any)
	return hasKey(phylo["phylo_group"].(map[string]map[string]any), "Non_tuberculosis_mycobacterium_complex")
}

func (p *AMRSpeciesPredictor) getPresentPhyloGroups(groups map[string]map[string]any, mixThreshold float64) map[string]map[string]any {
	if len(groups) == 0 {
		return groups
	}
	var high []string
	for pg, d := range groups {
		if asFloat(d["percent_coverage"]) > mixThreshold {
			high = append(high, pg)
		}
	}
	if len(high) > 1 {
		out := map[string]map[string]any{}
		for _, k := range high {
			out[k] = groups[k]
		}
		return out
	}
	return p.getBestCoverageDict(groups)
}

func (p *AMRSpeciesPredictor) getBestCoverageDict(cov map[string]map[string]any) map[string]map[string]any {
	if len(cov) == 0 {
		return cov
	}
	type item struct {
		name string
		data map[string]any
	}
	items := make([]item, 0, len(cov))
	for k, v := range cov {
		items = append(items, item{k, v})
	}
	sort.Slice(items, func(i, j int) bool {
		pi := asFloat(items[i].data["percent_coverage"])
		pj := asFloat(items[j].data["percent_coverage"])
		if pi == pj {
			return asFloat(items[i].data["median_depth"]) > asFloat(items[j].data["median_depth"])
		}
		return pi > pj
	})
	if asFloat(items[0].data["percent_coverage"]) > 0 {
		return map[string]map[string]any{items[0].name: items[0].data}
	}
	return map[string]map[string]any{}
}

func (p *AMRSpeciesPredictor) ChooseBest(phylogenetics map[string]map[string]map[string]any) map[string]map[string]map[string]any {
	phyloGroups := p.getPresentPhyloGroups(phylogenetics["phylo_group"], 50)
	phylogenetics["phylo_group"] = phyloGroups
	subComplexes := p.getPresentPhyloGroups(phylogenetics["sub_complex"], 90)
	phylogenetics["sub_complex"] = subComplexes

	species := map[string]map[string]any{}
	for pg := range phyloGroups {
		var speciesToConsider map[string]map[string]any
		if p.Hierarchy != nil && p.Hierarchy.Dict != nil && p.Hierarchy.Dict[pg] != nil {
			var allowed []string
			for _, subc := range p.Hierarchy.Dict[pg].Children {
				for speciesName := range subc.Children {
					allowed = append(allowed, speciesName)
				}
			}
			speciesToConsider = map[string]map[string]any{}
			for _, name := range allowed {
				if d, ok := phylogenetics["species"][name]; ok {
					speciesToConsider[name] = d
				} else {
					speciesToConsider[name] = map[string]any{"percent_coverage": 0}
				}
			}
		} else {
			speciesToConsider = phylogenetics["species"]
		}
		for k, v := range p.getPresentPhyloGroups(speciesToConsider, 90) {
			species[k] = v
		}
	}
	phylogenetics["species"] = species

	subSpecies := map[string]map[string]any{}
	for s := range species {
		var toConsider map[string]map[string]any
		if p.Hierarchy != nil && p.Hierarchy.GetChildren(s) != nil {
			children := p.Hierarchy.GetChildren(s)
			toConsider = map[string]map[string]any{}
			for k := range children {
				if d, ok := phylogenetics["lineage"][k]; ok {
					toConsider[k] = d
				} else {
					toConsider[k] = map[string]any{"percent_coverage": 0}
				}
			}
		} else {
			toConsider = phylogenetics["lineage"]
		}
		for k, v := range p.getBestCoverageDict(toConsider) {
			subSpecies[k] = v
		}
	}
	phylogenetics["lineage"] = subSpecies
	return phylogenetics
}

func DetectSpeciesAndGetDepths(set *CoverageSet, hierarchyJSONPath, wantedPhyloGroup string) (map[string]any, []float64, error) {
	if wantedPhyloGroup == "" {
		return map[string]any{}, nil, nil
	}

	phyloGroups := set.Groups["complex"]
	if len(phyloGroups) == 0 {
		phyloGroups = set.Groups["phylo_group"]
	}
	expectedDepth := calcExpectedDepth(phyloGroups)

	aggPhylo := aggregateTaxonCoverage(phyloGroups, expectedDepth, 5)
	aggSubComplex := aggregateTaxonCoverage(set.Groups["sub-complex"], expectedDepth, 50)
	aggSpecies := aggregateTaxonCoverage(set.Groups["species"], expectedDepth, 5)
	aggLineage := aggregateTaxonCoverage(set.Groups["sub-species"], expectedDepth, 5)

	p, err := NewAMRSpeciesPredictor(aggPhylo, aggSubComplex, aggSpecies, aggLineage, hierarchyJSONPath)
	if err != nil {
		return nil, nil, err
	}
	copyAllHitsUpwards(p.Hierarchy, aggPhylo, aggSubComplex, aggSpecies)

	phylogenetics := map[string]map[string]map[string]any{
		"phylo_group": aggPhylo,
		"sub_complex": aggSubComplex,
		"species":     aggSpecies,
		"lineage":     aggLineage,
	}
	phylogenetics = p.ChooseBest(phylogenetics)
	addUnknownWhereEmpty(phylogenetics["phylo_group"])
	addUnknownWhereEmpty(phylogenetics["sub_complex"])
	addUnknownWhereEmpty(phylogenetics["species"])
	addUnknownWhereEmpty(phylogenetics["lineage"])

	var depths []float64
	if d, ok := phylogenetics["phylo_group"][wantedPhyloGroup]; ok {
		depths = append(depths, asFloat(d["median_depth"]))
	}
	return map[string]any{
		"phylo_group": phylogenetics["phylo_group"],
		"sub_complex": phylogenetics["sub_complex"],
		"species":     phylogenetics["species"],
		"lineage":     phylogenetics["lineage"],
	}, depths, nil
}

func AddNCBINamesToPhylo(phylo map[string]any, ncbiNames map[string]string) {
	species, ok := phylo["species"].(map[string]map[string]any)
	if !ok || ncbiNames == nil {
		return
	}
	for name, d := range species {
		d["ncbi_names"] = ncbiNames[name]
		if d["ncbi_names"] == "" {
			d["ncbi_names"] = "UNKNOWN"
		}
	}
}

func calcExpectedDepth(groups map[string]*TaxonCoverage) float64 {
	if len(groups) == 0 {
		return 0
	}
	medians := make([]float64, 0)
	for _, cov := range groups {
		medians = append(medians, cov.Median...)
	}
	return medianFloat(medians)
}

func aggregateTaxonCoverage(groups map[string]*TaxonCoverage, expectedDepth, threshold float64) map[string]map[string]any {
	out := map[string]map[string]any{}
	for name, cov := range groups {
		totalPercentCovered := 0.0
		if cov.TotalBases > 0 {
			totalPercentCovered = round3(basesCovered(cov.PercentCoverage, cov.Length) / float64(cov.TotalBases))
		}
		medianDepth := medianFloat(cov.Median)
		minRequired := PercentCoverageFromExpectedCoverage(expectedDepth) * TaxonThresholds[name]
		if minRequired == 0 {
			minRequired = PercentCoverageFromExpectedCoverage(expectedDepth) * DefaultThreshold
		}
		filteredPC := cov.PercentCoverage
		filteredLen := cov.Length
		if totalPercentCovered < minRequired || medianDepth < 0.1*expectedDepth {
			filteredPC = nil
			filteredLen = nil
			filteredMedian := make([]float64, 0)
			for i, depth := range cov.Median {
				if depth > 0.1*expectedDepth {
					filteredPC = append(filteredPC, cov.PercentCoverage[i])
					filteredLen = append(filteredLen, cov.Length[i])
					filteredMedian = append(filteredMedian, depth)
				}
			}
			medianDepth = medianFloat(filteredMedian)
			if cov.TotalBases > 0 {
				totalPercentCovered = round3(basesCovered(filteredPC, filteredLen) / float64(cov.TotalBases))
			}
		}
		if totalPercentCovered > threshold {
			out[name] = map[string]any{
				"percent_coverage": totalPercentCovered,
				"median_depth":     medianDepth,
			}
		}
	}
	return out
}

func basesCovered(percentCoverage []float64, length []int) float64 {
	total := 0.0
	for i := range percentCoverage {
		total += percentCoverage[i] * float64(length[i])
	}
	return total
}

func copyAllHitsUpwards(h *Hierarchy, phyloGroupCovgs, subComplexCovgs, speciesCovgs map[string]map[string]any) {
	if h == nil {
		return
	}
	for complexName, node := range h.Dict {
		copyBestOneLevelUp(node.Children, subComplexCovgs, speciesCovgs)
		if _, ok := phyloGroupCovgs[complexName]; ok {
			continue
		}
	}
	copyBestOneLevelUp(h.Dict, phyloGroupCovgs, subComplexCovgs)
}

func copyBestOneLevelUp(hierarchy map[string]*HierarchyNode, parentCovgs, childCovgs map[string]map[string]any) {
	for parentName, node := range hierarchy {
		if _, ok := parentCovgs[parentName]; ok {
			continue
		}
		var best map[string]any
		for childName := range node.Children {
			cov, ok := childCovgs[childName]
			if !ok {
				continue
			}
			if best == nil || asFloat(cov["percent_coverage"]) > asFloat(best["percent_coverage"]) {
				best = map[string]any{
					"percent_coverage":    cov["percent_coverage"],
					"median_depth":        cov["median_depth"],
					"inferred_from_child": true,
				}
			}
		}
		if best != nil {
			parentCovgs[parentName] = best
		}
	}
}

func addUnknownWhereEmpty(covgs map[string]map[string]any) {
	if len(covgs) != 0 {
		return
	}
	covgs["Unknown"] = map[string]any{"percent_coverage": -1.0, "median_depth": -1.0}
}

func medianFloat(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sorted := append([]float64(nil), xs...)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func hasKey[K comparable, V any](m map[K]V, key K) bool {
	_, ok := m[key]
	return ok
}
