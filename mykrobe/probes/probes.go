package probes

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/martinghunt/faqt/seqio"
)

var varNameRe = regexp.MustCompile(`^([A-Z]+)([0-9]+)([A-Z]+)$`)

type Variant struct {
	Reference      string
	Start          int
	ReferenceBases string
	AlternateBases []string
}

func ParseVariantName(reference, varName string) (Variant, error) {
	m := varNameRe.FindStringSubmatch(varName)
	if m == nil {
		return Variant{}, fmt.Errorf("invalid variant name %q", varName)
	}
	start, err := strconvAtoi(m[2])
	if err != nil {
		return Variant{}, err
	}
	return Variant{
		Reference:      reference,
		Start:          start,
		ReferenceBases: m[1],
		AlternateBases: []string{m[3]},
	}, nil
}

func (v Variant) Length() int {
	return abs(len(v.AlternateBases[0]) - len(v.ReferenceBases))
}

func (v Variant) IsInsertion() bool {
	return len(v.AlternateBases[0]) > len(v.ReferenceBases)
}

func (v Variant) IsDeletion() bool {
	return len(v.AlternateBases[0]) < len(v.ReferenceBases)
}

func (v Variant) IsIndel() bool {
	return v.IsInsertion() || v.IsDeletion()
}

func (v Variant) Overlapping(other Variant) bool {
	start1, end1 := v.referenceSpan()
	start2, end2 := other.referenceSpan()
	return start1 <= end2 && start2 <= end1
}

func (v Variant) referenceSpan() (int, int) {
	end := v.Start + len(v.ReferenceBases) - 1
	if end < v.Start {
		end = v.Start
	}
	return v.Start, end
}

type Panel struct {
	Variant Variant
	Refs    []string
	Start   int
	Alts    []string
}

func newPanel(variant Variant, refs [][]byte, start int, alts [][]byte) Panel {
	refStrings := make([]string, 0, len(refs))
	for _, ref := range refs {
		refStrings = append(refStrings, string(ref))
	}
	altStrings := make([]string, 0, len(alts))
	for _, alt := range alts {
		altStrings = append(altStrings, string(alt))
	}
	refStrings = uniqueStrings(refStrings)
	altStrings = uniqueStrings(altStrings)
	refSet := make(map[string]struct{}, len(refStrings))
	for _, ref := range refStrings {
		refSet[ref] = struct{}{}
	}
	filteredAlts := altStrings[:0]
	for _, alt := range altStrings {
		if _, ok := refSet[alt]; ok {
			continue
		}
		filteredAlts = append(filteredAlts, alt)
	}
	refStrings = reorderRefsToAvoidOverlap(refStrings, filteredAlts)
	return Panel{Variant: variant, Refs: refStrings, Start: start, Alts: filteredAlts}
}

type Mutation struct {
	VarName           string
	Reference         string
	InputMutationName string
	ProteinCodingVar  bool
}

func (m Mutation) MutationOutputName() string {
	if m.InputMutationName != "" {
		return m.InputMutationName
	}
	return m.VarName
}

func (m Mutation) Variant() (Variant, error) {
	return ParseVariantName(m.Reference, m.VarName)
}

type LineageInfo struct {
	Name         string `json:"name"`
	UseRefAllele bool   `json:"use_ref_allele"`
	ReportName   string `json:"report_name,omitempty"`
}

func LoadDNAVarsTextFile(path, reference string) ([]Mutation, map[string]LineageInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	var mutations []Mutation
	lineages := map[string]LineageInfo{}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		if len(row) < 5 {
			return nil, nil, fmt.Errorf("expected at least 5 columns in %s", path)
		}
		geneName, pos, ref, alt := row[0], row[1], row[2], row[3]
		varName := geneName
		if geneName == "ref" {
			varName = ref + pos + alt
		}
		mutations = append(mutations, Mutation{Reference: reference, VarName: varName})
		if len(row) < 6 || row[5] == "" {
			continue
		}
		lineageName := row[5]
		useRefAllele := strings.HasPrefix(lineageName, "*")
		lineageName = strings.TrimPrefix(lineageName, "*")
		info := LineageInfo{Name: lineageName, UseRefAllele: useRefAllele}
		if len(row) > 6 && row[6] != "" {
			info.ReportName = row[6]
		}
		lineages[varName] = info
	}
	return mutations, lineages, nil
}

type AlleleGenerator struct {
	ReferencePath string
	Kmer          int
	ref           []byte
	referenceSeq  string
}

func NewAlleleGenerator(referencePath string, kmer int) (*AlleleGenerator, error) {
	ag := &AlleleGenerator{ReferencePath: referencePath, Kmer: kmer}
	if err := ag.readReference(); err != nil {
		return nil, err
	}
	return ag, nil
}

func (a *AlleleGenerator) readReference() error {
	reader, err := seqio.OpenPath(a.ReferencePath)
	if err != nil {
		return err
	}
	defer closeIfPossible(reader)
	var seq []byte
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		seq = append(seq, bytesToUpper(rec.Seq)...)
	}
	a.referenceSeq = string(seq)
	a.ref = append([]byte(nil), seq...)
	a.ref = append(a.ref, bytes.Repeat([]byte("N"), a.Kmer)...)
	return nil
}

func (a *AlleleGenerator) Create(v Variant, context []Variant) (Panel, error) {
	if err := a.checkValidVariant(v); err != nil {
		return Panel{}, err
	}
	context = a.removeOverlappingContexts(v, context)
	context = a.removeContextsNotWithinK(v, context)
	nullVariant := Variant{
		Reference:      v.Reference,
		Start:          v.Start,
		ReferenceBases: v.ReferenceBases,
		AlternateBases: []string{v.ReferenceBases},
	}
	refs, err := a.generateAlternatesOnAllBackgrounds(nullVariant, context)
	if err != nil {
		return Panel{}, err
	}
	alts, err := a.generateAlternatesOnAllBackgrounds(v, context)
	if err != nil {
		return Panel{}, err
	}
	if v.IsIndel() || hasIndel(context) {
		alts = a.trimUninformativeKmers(alts, refs)
	}
	return newPanel(v, refs, v.Start, alts), nil
}

func (a *AlleleGenerator) trimUninformativeKmers(alternates, references [][]byte) [][]byte {
	out := make([][]byte, 0, len(alternates))
	for i, altBytes := range alternates {
		alt := string(altBytes)
		ref := string(references[i])
		informative := informativeKmerIndexes(alt, a.Kmer, func(k string) bool { return !strings.Contains(ref, k) })
		if len(informative) > 0 {
			alt = alt[informative[0] : informative[len(informative)-1]+a.Kmer]
		}
		informative = informativeKmerIndexes(alt, a.Kmer, func(k string) bool { return !strings.Contains(a.referenceSeq, k) })
		if len(informative) > 0 {
			alt = alt[informative[0] : informative[len(informative)-1]+a.Kmer]
		}
		out = append(out, []byte(alt))
	}
	return out
}

func (a *AlleleGenerator) checkValidVariant(v Variant) error {
	index := v.Start - 1
	if len(v.AlternateBases) != 1 {
		return fmt.Errorf("probes can only be built for homozygous variants at this time")
	}
	ref := string(a.ref[index : index+len(v.ReferenceBases)])
	if ref != v.ReferenceBases {
		return fmt.Errorf("cannot create alleles as ref at pos %d is not %s (it's %s) are you sure you're using one-based co-ordinates?", v.Start, v.ReferenceBases, ref)
	}
	if v.Start <= 0 {
		return fmt.Errorf("position should be 1 based")
	}
	return nil
}

func (a *AlleleGenerator) removeOverlappingContexts(v Variant, context []Variant) []Variant {
	out := make([]Variant, 0, len(context))
	for _, c := range context {
		if !c.Overlapping(v) {
			out = append(out, c)
		}
	}
	return out
}

func (a *AlleleGenerator) removeContextsNotWithinK(v Variant, context []Variant) []Variant {
	out := make([]Variant, 0, len(context))
	for _, c := range context {
		effectivePos := c.Start
		if c.IsInsertion() {
			effectivePos = c.Start - c.Length()
		} else if c.IsDeletion() {
			effectivePos = c.Start + c.Length()
		}
		if abs(v.Start-effectivePos) < a.Kmer {
			out = append(out, c)
		}
	}
	return out
}

func (a *AlleleGenerator) generateAlternatesOnAllBackgrounds(v Variant, context []Variant) ([][]byte, error) {
	contextCombos := a.getAllContextCombinations(context)
	var alternates [][]byte
	for _, combo := range contextCombos {
		delta := a.calculateLengthDeltaFromIndels(v, combo)
		i, start, end := a.getStartEnd(v, delta)
		segment := append([]byte(nil), a.ref[start:end]...)
		background, err := a.generateBackgroundUsingContext(i, v, segment, append([]Variant(nil), combo...))
		if err != nil {
			names := make([]string, 0, len(combo)+1)
			for _, c := range combo {
				names = append(names, c.ReferenceBases+itoa(c.Start)+c.AlternateBases[0])
			}
			names = append(names, v.ReferenceBases+itoa(v.Start)+v.AlternateBases[0])
			return nil, fmt.Errorf("could not process context combo %s. %w", strings.Join(names, ","), err)
		}
		alt := append([]byte(nil), background...)
		i -= a.calculateLengthDeltaFromVariantList(filterVariants(combo, func(c Variant) bool { return c.Start <= v.Start && c.IsIndel() }))
		if string(alt[i:i+len(v.ReferenceBases)]) != v.ReferenceBases {
			return nil, fmt.Errorf("could not process context combo")
		}
		for _, altBase := range v.AlternateBases {
			candidate := append([]byte(nil), alt...)
			candidate = append(candidate[:i], append([]byte(altBase), candidate[i+len(v.ReferenceBases):]...)...)
			alternates = append(alternates, candidate)
		}
	}
	return alternates, nil
}

func (a *AlleleGenerator) getAllContextCombinations(context []Variant) [][]Variant {
	contexts := [][]Variant{{}}
	if len(context) == 0 {
		return contexts
	}
	for _, split := range a.createMultipleContexts(context) {
		contexts = append(contexts, combinationsOfBackgrounds(split)...)
	}
	return contexts
}

func (a *AlleleGenerator) createMultipleContexts(context []Variant) [][]Variant {
	return a.recursiveContextCreator([][]Variant{context})
}

func (a *AlleleGenerator) recursiveContextCreator(contexts [][]Variant) [][]Variant {
	compat := make([]bool, len(contexts))
	for i, context := range contexts {
		compat[i] = allVariantsCompatible(context)
	}
	if allTrue(compat) {
		return contexts
	}
	i := slices.Index(compat, false)
	incompatible := contexts[i]
	contexts = append(append([][]Variant{}, contexts[:i]...), contexts[i+1:]...)
	contexts = append(contexts, splitContext(incompatible)...)
	return a.recursiveContextCreator(contexts)
}

func (a *AlleleGenerator) getStartEnd(v Variant, delta int) (int, int, int) {
	shift := 0
	kmer := a.Kmer
	if len(v.ReferenceBases) > 2*kmer {
		kmer = int(math.Ceil(float64(len(v.ReferenceBases))/2.0)) + 5
	} else if v.Length() > 2*kmer {
		kmer = int(math.Ceil(float64(v.Length())/2.0)) + 5
	}
	if len(v.ReferenceBases) > kmer {
		shift = (kmer - 1) - int(math.Floor(float64((2*kmer+1)-len(v.ReferenceBases))/2.0))
	}
	startDelta := int(math.Floor(float64(delta) / 2.0))
	endDelta := int(math.Ceil(float64(delta) / 2.0))
	startIndex := v.Start - kmer - startDelta
	endIndex := v.Start + kmer + endDelta - 1
	minProbeLength := (2 * kmer) - 1
	i := kmer - 1 + startDelta
	if startIndex < 0 {
		diff := abs(startIndex)
		startIndex = 0
		endIndex += diff
		i -= diff
	}
	startIndex += shift
	endIndex += shift
	i -= shift
	if endIndex-startIndex >= minProbeLength {
		return i, startIndex, endIndex
	}
	return a.getStartEnd(v, 0)
}

func (a *AlleleGenerator) calculateLengthDeltaFromIndels(v Variant, context []Variant) int {
	vars := append(append([]Variant{}, context...), v)
	return a.calculateLengthDeltaFromVariantList(vars)
}

func (a *AlleleGenerator) calculateLengthDeltaFromVariantList(vars []Variant) int {
	deletions := 0
	insertions := 0
	for _, v := range vars {
		if v.IsDeletion() {
			deletions += v.Length()
		}
		if v.IsInsertion() {
			insertions += v.Length()
		}
	}
	delta := deletions - insertions
	if delta < -a.Kmer {
		return abs(delta)
	}
	return delta
}

func (a *AlleleGenerator) generateBackgroundUsingContext(i int, v Variant, segment []byte, context []Variant) ([]byte, error) {
	background := append([]byte(nil), segment...)
	var added []Variant
	for _, variant := range context {
		for _, alt := range variant.AlternateBases {
			j := i + variant.Start - v.Start
			j -= a.calculateLengthDeltaFromVariantList(filterVariants(added, func(c Variant) bool { return c.Start <= variant.Start && c.IsIndel() }))
			if j > len(background) || j < 0 {
				continue
			}
			if j+len(variant.ReferenceBases) > len(background) {
				hang := j + len(variant.ReferenceBases) - len(background)
				if string(background[j:len(background)]) != variant.ReferenceBases[:len(variant.ReferenceBases)-hang] {
					return nil, fmt.Errorf("could not process variant")
				}
				background = append(background[:j], append([]byte(alt[:len(variant.ReferenceBases)-hang]), background[len(background):]...)...)
			} else {
				if string(background[j:j+len(variant.ReferenceBases)]) != variant.ReferenceBases {
					return nil, fmt.Errorf("could not process variant")
				}
				background = append(background[:j], append([]byte(alt), background[j+len(variant.ReferenceBases):]...)...)
			}
			added = append(added, variant)
		}
	}
	return background, nil
}

func splitContext(context []Variant) [][]Variant {
	v1, v2 := firstTwoIncompatibleVariants(context)
	var nov1, nov2 []Variant
	for _, x := range context {
		if !sameVariant(x, v1) {
			nov1 = append(nov1, x)
		}
		if !sameVariant(x, v2) {
			nov2 = append(nov2, x)
		}
	}
	return [][]Variant{nov1, nov2}
}

func firstTwoIncompatibleVariants(context []Variant) (Variant, Variant) {
	for i := 0; i < len(context); i++ {
		for j := i + 1; j < len(context); j++ {
			if context[i].Overlapping(context[j]) {
				return context[i], context[j]
			}
		}
	}
	return Variant{}, Variant{}
}

func allVariantsCompatible(vars []Variant) bool {
	for i := 0; i < len(vars); i++ {
		for j := i + 1; j < len(vars); j++ {
			if vars[i].Overlapping(vars[j]) {
				return false
			}
		}
	}
	return true
}

func sameVariant(a, b Variant) bool {
	if a.Reference != b.Reference || a.Start != b.Start || a.ReferenceBases != b.ReferenceBases {
		return false
	}
	return slices.Equal(a.AlternateBases, b.AlternateBases)
}

func combinationsOfBackgrounds(context []Variant) [][]Variant {
	var out [][]Variant
	for l := 1; l <= len(context); l++ {
		var walk func(start int, current []Variant)
		walk = func(start int, current []Variant) {
			if len(current) == l {
				out = append(out, append([]Variant(nil), current...))
				return
			}
			for i := start; i < len(context); i++ {
				walk(i+1, append(current, context[i]))
			}
		}
		walk(0, nil)
	}
	return out
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func reorderRefsToAvoidOverlap(refs, alts []string) []string {
	if len(refs) != len(alts) || len(refs) <= 1 {
		return refs
	}
	alreadyGood := true
	for i := range refs {
		if hasOverlappingKmers(refs[i], alts[i], 31) {
			alreadyGood = false
			break
		}
	}
	if alreadyGood {
		return refs
	}
	edges := make([][]int, len(alts))
	for ai, alt := range alts {
		for ri, ref := range refs {
			if !hasOverlappingKmers(ref, alt, 31) {
				edges[ai] = append(edges[ai], ri)
			}
		}
		if len(edges[ai]) == 0 {
			return refs
		}
	}
	matchAltToRef := make([]int, len(alts))
	for i := range matchAltToRef {
		matchAltToRef[i] = -1
	}
	used := make([]bool, len(refs))
	var assign func(int) bool
	assign = func(ai int) bool {
		if ai == len(alts) {
			return true
		}
		for _, ri := range edges[ai] {
			if used[ri] {
				continue
			}
			used[ri] = true
			matchAltToRef[ai] = ri
			if assign(ai + 1) {
				return true
			}
			matchAltToRef[ai] = -1
			used[ri] = false
		}
		return false
	}
	if !assign(0) {
		return refs
	}
	out := make([]string, len(refs))
	for ai, ri := range matchAltToRef {
		out[ai] = refs[ri]
	}
	return out
}

func hasOverlappingKmers(ref, alt string, k int) bool {
	if len(ref) < k || len(alt) < k {
		return false
	}
	seen := make(map[string]struct{}, len(ref)-k+1)
	for i := 0; i <= len(ref)-k; i++ {
		seen[ref[i:i+k]] = struct{}{}
	}
	for i := 0; i <= len(alt)-k; i++ {
		if _, ok := seen[alt[i:i+k]]; ok {
			return true
		}
	}
	return false
}

func informativeKmerIndexes(seq string, k int, keep func(string) bool) []int {
	if len(seq) < k {
		return nil
	}
	var out []int
	for i := 0; i <= len(seq)-k; i++ {
		if keep(seq[i : i+k]) {
			out = append(out, i)
		}
	}
	return out
}

func filterVariants(vars []Variant, keep func(Variant) bool) []Variant {
	out := make([]Variant, 0, len(vars))
	for _, v := range vars {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

func hasIndel(vars []Variant) bool {
	for _, v := range vars {
		if v.IsIndel() {
			return true
		}
	}
	return false
}

func allTrue(items []bool) bool {
	for _, ok := range items {
		if !ok {
			return false
		}
	}
	return true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func bytesToUpper(seq []byte) []byte {
	out := make([]byte, len(seq))
	for i, b := range seq {
		if b >= 'a' && b <= 'z' {
			out[i] = b - 32
		} else {
			out[i] = b
		}
	}
	return out
}

func closeIfPossible(v any) {
	if c, ok := v.(io.Closer); ok {
		_ = c.Close()
	}
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func strconvAtoi(s string) (int, error) {
	var n int
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid integer %q", s)
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

func DefaultReferenceName(path string) string {
	return strings.Split(filepath.Base(path), ".fa")[0]
}
