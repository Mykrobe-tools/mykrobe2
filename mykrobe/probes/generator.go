package probes

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"slices"
	"strings"

	"github.com/martinghunt/faqt/seqio"
)

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
