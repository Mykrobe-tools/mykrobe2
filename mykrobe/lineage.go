package mykrobe

import (
	"slices"
	"strings"
)

type LineageVariant struct {
	Name         string `json:"name"`
	UseRefAllele bool   `json:"use_ref_allele"`
	ReportName   string `json:"report_name,omitempty"`
}

type lineageNode struct {
	Name     string
	Parent   *lineageNode
	Children []*lineageNode
}

type LineagePredictor struct {
	VariantToLineage map[string]LineageVariant
	ReportNames      map[string]string
	TreeRoot         *lineageNode
	TreeNodes        map[string]*lineageNode
}

func NewLineagePredictor(variantToLineage map[string]LineageVariant) *LineagePredictor {
	p := &LineagePredictor{
		VariantToLineage: variantToLineage,
		ReportNames:      map[string]string{},
		TreeRoot:         &lineageNode{Name: "root"},
		TreeNodes:        map[string]*lineageNode{},
	}
	for _, d := range variantToLineage {
		if d.ReportName != "" {
			p.ReportNames[d.Name] = d.ReportName
		} else {
			p.ReportNames[d.Name] = d.Name
		}
	}
	p.makeTree()
	return p
}

func LineageToSelfPlusParents(lineage string) []string {
	pieces := strings.Split(lineage, ".")
	out := make([]string, len(pieces))
	for i := range pieces {
		out[i] = strings.Join(pieces[:i+1], ".")
	}
	return out
}

func (p *LineagePredictor) makeTree() {
	for _, lineage := range p.VariantToLineage {
		nodes := LineageToSelfPlusParents(lineage.Name)
		for i, name := range nodes {
			if _, ok := p.TreeNodes[name]; ok {
				continue
			}
			parent := p.TreeRoot
			if i > 0 {
				parent = p.TreeNodes[nodes[i-1]]
			}
			node := &lineageNode{Name: name, Parent: parent}
			parent.Children = append(parent.Children, node)
			slices.SortFunc(parent.Children, func(a, b *lineageNode) int {
				return strings.Compare(a.Name, b.Name)
			})
			p.TreeNodes[name] = node
		}
	}
}

func (n *lineageNode) leaves() []*lineageNode {
	if len(n.Children) == 0 {
		return []*lineageNode{n}
	}
	var out []*lineageNode
	for _, child := range n.Children {
		out = append(out, child.leaves()...)
	}
	return out
}

func (n *lineageNode) pathFromRoot() []*lineageNode {
	if n.Parent == nil {
		return []*lineageNode{n}
	}
	return append(n.Parent.pathFromRoot(), n)
}

func (p *LineagePredictor) ScoreEachLineageNode(lineageCalls map[string]map[string]Call) map[string]float64 {
	scores := map[string]float64{}
	for lineageName, varDict := range lineageCalls {
		var best *float64
		for varName, call := range varDict {
			wanted := []int{1, 1}
			if p.VariantToLineage[varName].UseRefAllele {
				wanted = []int{0, 0}
			}
			multiplier := -1.0
			if slices.Equal(call.Genotype, wanted) {
				filters, _ := call.Info["filter"].([]string)
				if len(filters) == 0 {
					multiplier = 1
				} else {
					multiplier = 0.5
				}
			} else if slices.Equal(call.Genotype, []int{0, 1}) {
				multiplier = 0.5
			}
			score := multiplier * asFloat(call.Info["conf"])
			if best == nil || *best < score {
				x := score
				best = &x
			}
		}
		if best == nil {
			scores[lineageName] = 0
		} else {
			scores[lineageName] = *best
		}
	}
	return scores
}

type PathScore struct {
	Score   float64            `json:"score"`
	Lineage string             `json:"lineage"`
	Scores  map[string]float64 `json:"scores"`
}

func (p *LineagePredictor) GetPathsAndScores(lineageCalls map[string]map[string]Call) []PathScore {
	var paths []PathScore
	used := map[string]struct{}{}
	nodeScores := p.ScoreEachLineageNode(lineageCalls)
	for _, leaf := range p.TreeRoot.leaves() {
		pathNodes := leaf.pathFromRoot()[1:]
		type pair struct {
			name  string
			score float64
		}
		pathScores := make([]pair, len(pathNodes))
		for i, n := range pathNodes {
			pathScores[i] = pair{n.Name, nodeScores[n.Name]}
		}
		pathLeaf := leaf
		for len(pathScores) > 0 && pathScores[len(pathScores)-1].score <= 0 {
			pathScores = pathScores[:len(pathScores)-1]
			pathLeaf = pathLeaf.Parent
		}
		if len(pathScores) == 0 {
			continue
		}
		if _, ok := used[pathLeaf.Name]; ok {
			continue
		}
		scores := map[string]float64{}
		total := 0.0
		for _, ps := range pathScores {
			scores[ps.name] = ps.score
			total += ps.score
		}
		paths = append(paths, PathScore{Score: total, Lineage: pathLeaf.Name, Scores: scores})
		used[pathLeaf.Name] = struct{}{}
	}
	slices.SortFunc(paths, func(a, b PathScore) int {
		if a.Score == b.Score {
			return strings.Compare(a.Lineage, b.Lineage)
		}
		if a.Score > b.Score {
			return -1
		}
		return 1
	})
	return paths
}

func (p *LineagePredictor) CallLineageUsingConfScores(lineageCalls map[string]map[string]Call) map[string]any {
	paths := p.GetPathsAndScores(lineageCalls)
	if len(paths) == 0 {
		return nil
	}
	bestScore := paths[0].Score
	bestPaths := []PathScore{}
	for _, ps := range paths {
		if ps.Score == bestScore {
			bestPaths = append(bestPaths, ps)
		}
	}
	result := map[string]any{
		"lineage": []string{},
		"calls":   map[string]map[string]map[string]Call{},
	}
	for _, path := range bestPaths {
		result["lineage"] = append(result["lineage"].([]string), path.Lineage)
		used := map[string]map[string]Call{}
		for lineage := range path.Scores {
			used[lineage] = lineageCalls[lineage]
		}
		result["calls"].(map[string]map[string]map[string]Call)[path.Lineage] = used
	}
	return result
}

func (p *LineagePredictor) GenotypeEachLineageNode(lineageCalls map[string]map[string]Call) map[string]float64 {
	genotypes := map[string]float64{}
	for lineageName, varDict := range lineageCalls {
		var best *float64
		for varName, call := range varDict {
			wanted := []int{1, 1}
			if p.VariantToLineage[varName].UseRefAllele {
				wanted = []int{0, 0}
			}
			geno := 0.0
			if slices.Equal(call.Genotype, wanted) {
				geno = 1
			} else if slices.Equal(call.Genotype, []int{0, 1}) {
				geno = 0.5
			}
			if best == nil || *best < geno {
				x := geno
				best = &x
			}
		}
		if best == nil {
			genotypes[lineageName] = 0
		} else {
			genotypes[lineageName] = *best
		}
	}
	return genotypes
}

type GoodPath struct {
	GoodNodes int                `json:"good_nodes"`
	TreeDepth int                `json:"tree_depth"`
	Genotypes map[string]float64 `json:"genotypes"`
}

func (p *LineagePredictor) GetGoodPathsUsingGenotypeCalls(lineageCalls map[string]map[string]Call, minFracCalled float64) map[string]GoodPath {
	paths := map[string]GoodPath{}
	nodeGenos := p.GenotypeEachLineageNode(lineageCalls)
	for _, leaf := range p.TreeRoot.leaves() {
		pathNodes := leaf.pathFromRoot()[1:]
		type pair struct {
			name string
			geno float64
		}
		pathGenos := make([]pair, len(pathNodes))
		for i, n := range pathNodes {
			pathGenos[i] = pair{n.Name, nodeGenos[n.Name]}
		}
		pathLeaf := leaf
		for len(pathGenos) > 0 && pathGenos[len(pathGenos)-1].geno == 0 {
			pathGenos = pathGenos[:len(pathGenos)-1]
			pathLeaf = pathLeaf.Parent
		}
		numGood := 0
		for _, pg := range pathGenos {
			if pg.geno > 0 {
				numGood++
			}
		}
		if len(pathGenos) == 0 || float64(numGood)/float64(len(pathGenos)) <= minFracCalled {
			continue
		}
		if _, ok := paths[pathLeaf.Name]; ok {
			continue
		}
		skip := false
		for name := range paths {
			if strings.Contains(name, pathLeaf.Name) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		for name := range paths {
			if strings.Contains(pathLeaf.Name, name) {
				delete(paths, name)
			}
		}
		genos := map[string]float64{}
		for _, pg := range pathGenos {
			genos[pg.name] = pg.geno
		}
		paths[pathLeaf.Name] = GoodPath{GoodNodes: numGood, TreeDepth: len(pathGenos), Genotypes: genos}
	}
	return paths
}

func (p *LineagePredictor) CallLineage(lineageCalls map[string]map[string]Call, minFracCalled float64) map[string]any {
	paths := p.GetGoodPathsUsingGenotypeCalls(lineageCalls, minFracCalled)
	if len(paths) == 0 {
		return nil
	}
	lineages := make([]string, 0, len(paths))
	for lineage := range paths {
		lineages = append(lineages, lineage)
	}
	slices.Sort(lineages)
	calls := map[string]map[string]map[string]Call{}
	for _, lineage := range lineages {
		calls[lineage] = map[string]map[string]Call{}
		for lineage2 := range paths[lineage].Genotypes {
			if lc, ok := lineageCalls[lineage2]; ok {
				calls[lineage][lineage2] = lc
			}
		}
	}
	result := map[string]any{
		"lineage":       lineages,
		"calls_summary": paths,
		"calls":         calls,
	}
	p.applyReportNames(result)
	return result
}

func (p *LineagePredictor) applyReportNames(result map[string]any) {
	lineages := result["lineage"].([]string)
	for i, x := range lineages {
		if y, ok := p.ReportNames[x]; ok {
			lineages[i] = y
		}
	}
	result["lineage"] = lineages

	oldCalls := result["calls"].(map[string]map[string]map[string]Call)
	newCalls := map[string]map[string]map[string]Call{}
	for lineage, d := range oldCalls {
		lineage2 := p.ReportNames[lineage]
		newCalls[lineage2] = p.replaceNestedCallKeys(d)
	}
	result["calls"] = newCalls

	oldSummary := result["calls_summary"].(map[string]GoodPath)
	newSummary := map[string]GoodPath{}
	for lineage, d := range oldSummary {
		d.Genotypes = p.replaceFloatKeys(d.Genotypes)
		newSummary[p.ReportNames[lineage]] = d
	}
	result["calls_summary"] = newSummary
}

func (p *LineagePredictor) replaceNestedCallKeys(d map[string]map[string]Call) map[string]map[string]Call {
	out := map[string]map[string]Call{}
	for k, v := range d {
		if name, ok := p.ReportNames[k]; ok {
			out[name] = v
		} else {
			out[k] = v
		}
	}
	return out
}

func (p *LineagePredictor) replaceFloatKeys(d map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for k, v := range d {
		if name, ok := p.ReportNames[k]; ok {
			out[name] = v
		} else {
			out[k] = v
		}
	}
	return out
}

func asFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	default:
		return 0
	}
}
