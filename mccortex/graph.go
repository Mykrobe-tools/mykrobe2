package mccortex

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/martinghunt/faqt/seqio"
)

type Graph struct {
	K      int
	Counts map[string]uint32
	Succ   map[string]map[string]struct{}
	Pred   map[string]map[string]struct{}
}

func NewGraph(k int) (*Graph, error) {
	if k < 1 || k > 31 {
		return nil, fmt.Errorf("k must be in range 1..31, got %d", k)
	}
	return &Graph{
		K:      k,
		Counts: make(map[string]uint32),
		Succ:   make(map[string]map[string]struct{}),
		Pred:   make(map[string]map[string]struct{}),
	}, nil
}

func LoadGraph(path string) (*Graph, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var disk graphDisk
	if err := json.NewDecoder(f).Decode(&disk); err != nil {
		return nil, err
	}

	g, err := NewGraph(disk.K)
	if err != nil {
		return nil, err
	}
	g.Counts = make(map[string]uint32, len(disk.Counts))
	for k, v := range disk.Counts {
		g.Counts[k] = v
	}
	for _, edge := range disk.Edges {
		g.addEdge(edge.From, edge.To)
	}
	return g, nil
}

func (g *Graph) Save(path string) error {
	edges := make([]graphEdge, 0)
	for from, succs := range g.Succ {
		for to := range succs {
			edges = append(edges, graphEdge{From: from, To: to})
		}
	}
	slices.SortFunc(edges, func(a, b graphEdge) int {
		if a.From != b.From {
			return strings.Compare(a.From, b.From)
		}
		return strings.Compare(a.To, b.To)
	})

	disk := graphDisk{
		Format: "mykrobe2-graph-v1",
		K:      g.K,
		Counts: g.Counts,
		Edges:  edges,
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(disk)
}

func (g *Graph) AddPath(path string) error {
	reader, err := seqio.OpenPath(path)
	if err != nil {
		return err
	}
	defer closeIfPossible(reader)

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		g.AddSequence(rec.Seq)
	}
}

func (g *Graph) AddSequence(seq []byte) {
	if len(seq) < g.K {
		return
	}
	upper := strings.ToUpper(string(seq))
	var prev string

	for i := 0; i+g.K <= len(upper); i++ {
		kmer := upper[i : i+g.K]
		if !validDNA(kmer) {
			prev = ""
			continue
		}

		canon := canonicalString(kmer)
		g.Counts[canon]++

		if prev != "" {
			g.addBiEdge(prev, kmer)
		}
		prev = kmer
	}
}

func (g *Graph) KmerCount(kmer string) uint32 {
	return g.Counts[canonicalString(strings.ToUpper(kmer))]
}

func (g *Graph) NumKmers() int {
	return len(g.Counts)
}

func (g *Graph) Unitigs() []string {
	visited := make(map[string]bool)
	unitigs := make(map[string]struct{})
	covered := make(map[string]bool)
	nodes := sortedKeys(g.Counts)

	for _, node := range nodes {
		for _, orient := range []string{node, revcomp(node)} {
			if visited[orient] {
				continue
			}
			indeg := len(g.Pred[orient])
			outdeg := len(g.Succ[orient])
			if outdeg == 0 {
				visited[orient] = true
				continue
			}
			if indeg == 1 && outdeg == 1 {
				continue
			}
			for _, next := range sortedSet(g.Succ[orient]) {
				path := []string{orient, next}
				visited[orient] = true
				visited[revcomp(orient)] = true
				curr := next
				for {
					visited[curr] = true
					visited[revcomp(curr)] = true
					if len(g.Pred[curr]) != 1 || len(g.Succ[curr]) != 1 {
						break
					}
					nxt := onlyMember(g.Succ[curr])
					if nxt == orient {
						break
					}
					path = append(path, nxt)
					curr = nxt
				}
				seq := canonicalUnitig(pathToSequence(path))
				unitigs[seq] = struct{}{}
				for _, k := range unitigCanonicals(seq, g.K) {
					covered[k] = true
				}
			}
		}
	}

	for _, node := range nodes {
		for _, orient := range []string{node, revcomp(node)} {
			if visited[orient] {
				continue
			}
			path := []string{orient}
			visited[orient] = true
			visited[revcomp(orient)] = true
			curr := orient
			for {
				next := onlyMember(g.Succ[curr])
				if next == orient {
					break
				}
				path = append(path, next)
				curr = next
				visited[curr] = true
				visited[revcomp(curr)] = true
			}
			seq := canonicalUnitig(pathToSequence(path))
			if sameCanonicalPath(path) {
				seq = canonicalString(path[0])
			}
			unitigs[seq] = struct{}{}
			for _, k := range unitigCanonicals(seq, g.K) {
				covered[k] = true
			}
		}
	}

	for _, node := range nodes {
		if covered[node] {
			continue
		}
		unitigs[node] = struct{}{}
	}

	out := sortedKeys(unitigs)
	return out
}

func (g *Graph) Join(others ...*Graph) (*Graph, error) {
	out, err := NewGraph(g.K)
	if err != nil {
		return nil, err
	}
	graphs := append([]*Graph{g}, others...)
	for _, graph := range graphs {
		if graph.K != g.K {
			return nil, fmt.Errorf("cannot join graphs with different k: %d vs %d", g.K, graph.K)
		}
		for kmer, count := range graph.Counts {
			out.Counts[kmer] += count
		}
		for from, succs := range graph.Succ {
			for to := range succs {
				out.addEdge(from, to)
			}
		}
	}
	return out, nil
}

func (g *Graph) Intersect(graphs ...*Graph) (*Graph, error) {
	if len(graphs) == 0 {
		return g.Clone(), nil
	}
	keep := make(map[string]bool, len(g.Counts))
	for k := range g.Counts {
		keep[k] = true
	}
	for _, graph := range graphs {
		if graph.K != g.K {
			return nil, fmt.Errorf("cannot intersect graphs with different k: %d vs %d", g.K, graph.K)
		}
		for k := range keep {
			if _, ok := graph.Counts[k]; !ok {
				delete(keep, k)
			}
		}
	}
	return g.FilterCanonical(keep), nil
}

func (g *Graph) Clone() *Graph {
	out, _ := NewGraph(g.K)
	for kmer, count := range g.Counts {
		out.Counts[kmer] = count
	}
	for from, succs := range g.Succ {
		for to := range succs {
			out.addEdge(from, to)
		}
	}
	return out
}

func (g *Graph) SubgraphFromSequences(seqs [][]byte, dist int, invert, wholeUnitigs bool) (*Graph, error) {
	selected := make(map[string]bool)
	for _, seq := range seqs {
		selectedSeed := g.seedCanonicals(seq)
		for k := range selectedSeed {
			selected[k] = true
		}
	}

	selected = g.expandByDistance(selected, dist)
	if wholeUnitigs && len(selected) > 0 {
		selected = g.expandToWholeUnitigs(selected)
	}
	if invert {
		inv := make(map[string]bool, len(g.Counts))
		for k := range g.Counts {
			if !selected[k] {
				inv[k] = true
			}
		}
		selected = inv
	}

	return g.FilterCanonical(selected), nil
}

func (g *Graph) FilterCanonical(keep map[string]bool) *Graph {
	out, _ := NewGraph(g.K)
	for k, count := range g.Counts {
		if keep[k] {
			out.Counts[k] = count
		}
	}
	for from, succs := range g.Succ {
		if !keep[canonicalString(from)] {
			continue
		}
		for to := range succs {
			if keep[canonicalString(to)] {
				out.addEdge(from, to)
			}
		}
	}
	return out
}

func (g *Graph) UnitigsFASTA() string {
	var b strings.Builder
	for i, seq := range g.Unitigs() {
		fmt.Fprintf(&b, ">unitig%d\n%s\n", i, seq)
	}
	return b.String()
}

func (g *Graph) Kmers() []string {
	return sortedKeys(g.Counts)
}

func (g *Graph) SummarizePanelPath(path string) ([]CoverageSummary, error) {
	reader, err := seqio.OpenPath(path)
	if err != nil {
		return nil, err
	}
	defer closeIfPossible(reader)

	var out []CoverageSummary
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		summary, err := g.SummarizeSequence(rec.Name, rec.Seq)
		if err != nil {
			return nil, err
		}
		out = append(out, summary)
	}
}

func (g *Graph) SummarizeSequence(name string, seq []byte) (CoverageSummary, error) {
	klen := 0
	if len(seq) >= g.K {
		klen = len(seq) - g.K + 1
	}

	summary := CoverageSummary{Name: name, Colour: 0, KmerLength: klen}
	if klen <= 1 {
		return summary, nil
	}

	covgs := make([]uint32, klen)
	upper := strings.ToUpper(string(seq))
	for i := 1; i < klen; i++ {
		kmer := upper[i : i+g.K]
		if !validDNA(kmer) {
			continue
		}
		covgs[i] = g.Counts[canonicalString(kmer)]
	}

	minDepth := uint32(^uint32(0))
	var nonZero float64
	for i := 1; i < klen; i++ {
		depth := covgs[i]
		summary.KmerCount += depth
		if depth == 0 {
			continue
		}
		nonZero++
		if depth < minDepth {
			minDepth = depth
		}
	}
	if minDepth == ^uint32(0) {
		minDepth = 0
	}

	sorted := slices.Clone(covgs)
	slices.Sort(sorted)
	summary.MedianDepth = medianUint32(sorted)
	summary.MinDepth = minDepth
	summary.PercentCoverage = nonZero / float64(klen-1)
	return summary, nil
}

func (g *Graph) addBiEdge(from, to string) {
	g.addEdge(from, to)
	g.addEdge(revcomp(to), revcomp(from))
}

func (g *Graph) addEdge(from, to string) {
	if g.Succ[from] == nil {
		g.Succ[from] = make(map[string]struct{})
	}
	if g.Pred[to] == nil {
		g.Pred[to] = make(map[string]struct{})
	}
	g.Succ[from][to] = struct{}{}
	g.Pred[to][from] = struct{}{}
}

func (g *Graph) seedCanonicals(seq []byte) map[string]bool {
	out := make(map[string]bool)
	upper := strings.ToUpper(string(seq))
	for i := 0; i+g.K <= len(upper); i++ {
		kmer := upper[i : i+g.K]
		if !validDNA(kmer) {
			continue
		}
		canon := canonicalString(kmer)
		if _, ok := g.Counts[canon]; ok {
			out[canon] = true
		}
	}
	return out
}

func (g *Graph) expandByDistance(seed map[string]bool, dist int) map[string]bool {
	if len(seed) == 0 {
		return map[string]bool{}
	}
	out := make(map[string]bool, len(seed))
	type item struct {
		kmer  string
		depth int
	}
	queue := make([]item, 0, len(seed))
	for k := range seed {
		out[k] = true
		queue = append(queue, item{kmer: k, depth: 0})
	}
	for head := 0; head < len(queue); head++ {
		it := queue[head]
		if it.depth >= dist {
			continue
		}
		for _, neigh := range g.neighborCanonicals(it.kmer) {
			if out[neigh] {
				continue
			}
			out[neigh] = true
			queue = append(queue, item{kmer: neigh, depth: it.depth + 1})
		}
	}
	return out
}

func (g *Graph) neighborCanonicals(canon string) []string {
	set := make(map[string]struct{})
	for _, orient := range []string{canon, revcomp(canon)} {
		for from := range g.Pred[orient] {
			set[canonicalString(from)] = struct{}{}
		}
		for to := range g.Succ[orient] {
			set[canonicalString(to)] = struct{}{}
		}
	}
	delete(set, canon)
	return sortedSet(set)
}

func (g *Graph) expandToWholeUnitigs(seed map[string]bool) map[string]bool {
	out := make(map[string]bool, len(seed))
	for k := range seed {
		out[k] = true
	}
	for _, unitig := range g.Unitigs() {
		kmers := unitigCanonicals(unitig, g.K)
		touched := false
		for _, k := range kmers {
			if seed[k] {
				touched = true
				break
			}
		}
		if touched {
			for _, k := range kmers {
				out[k] = true
			}
		}
	}
	return out
}

type graphDisk struct {
	Format string            `json:"format"`
	K      int               `json:"k"`
	Counts map[string]uint32 `json:"counts"`
	Edges  []graphEdge       `json:"edges"`
}

type graphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func validDNA(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 'A', 'C', 'G', 'T':
		default:
			return false
		}
	}
	return true
}

func canonicalString(s string) string {
	rc := revcomp(s)
	if rc < s {
		return rc
	}
	return s
}

func canonicalUnitig(s string) string {
	rc := revcomp(s)
	if rc < s {
		return rc
	}
	return s
}

func revcomp(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		switch s[len(s)-1-i] {
		case 'A':
			b[i] = 'T'
		case 'C':
			b[i] = 'G'
		case 'G':
			b[i] = 'C'
		case 'T':
			b[i] = 'A'
		case 'a':
			b[i] = 't'
		case 'c':
			b[i] = 'g'
		case 'g':
			b[i] = 'c'
		case 't':
			b[i] = 'a'
		default:
			b[i] = 'N'
		}
	}
	return strings.ToUpper(string(b))
}

func pathToSequence(path []string) string {
	if len(path) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(path[0])
	for i := 1; i < len(path); i++ {
		b.WriteByte(path[i][len(path[i])-1])
	}
	return b.String()
}

func unitigCanonicals(seq string, k int) []string {
	if len(seq) < k {
		return nil
	}
	out := make([]string, 0, len(seq)-k+1)
	for i := 0; i+k <= len(seq); i++ {
		out = append(out, canonicalString(seq[i:i+k]))
	}
	return out
}

func sameCanonicalPath(path []string) bool {
	if len(path) == 0 {
		return false
	}
	first := canonicalString(path[0])
	for i := 1; i < len(path); i++ {
		if canonicalString(path[i]) != first {
			return false
		}
	}
	return true
}

func sortedKeys[M ~map[string]V, V any](m M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func sortedSet(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	return sortedKeys(m)
}

func onlyMember(m map[string]struct{}) string {
	for k := range m {
		return k
	}
	return ""
}
