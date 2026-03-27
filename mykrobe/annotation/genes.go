package annotation

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/martinghunt/faqt/seqio"
	"github.com/martinghunt/mykrobe2/mykrobe"
)

var splitVarRe = regexp.MustCompile(`^([A-Z]+)([-0-9]+)([A-Z/*]+)$`)
var locationRe = regexp.MustCompile(`^(complement\()?([0-9]+)\.\.([0-9]+)\)?$`)

type Region struct {
	Reference string
	Start     int
	End       int
	Forward   bool
}

func (r Region) Strand() string {
	if r.Forward {
		return "forward"
	}
	return "reverse"
}

func (r Region) Seq() string {
	seq := r.Reference[r.Start-1 : r.End]
	if r.Forward {
		return seq
	}
	return mykrobe.RevcompDNA(seq)
}

func (r Region) GetReferencePosition(pos int) (int, error) {
	switch {
	case pos < 0 && r.Forward:
		return r.Start + pos, nil
	case pos < 0 && !r.Forward:
		return r.End - pos, nil
	case pos > 0 && r.Forward:
		return r.Start + pos - 1, nil
	case pos > 0 && !r.Forward:
		return r.End - pos + 1, nil
	default:
		return 0, fmt.Errorf("positions are 1-based")
	}
}

type Gene struct {
	Name string
	Region
}

func (g Gene) Prot() string {
	prot := mykrobe.TranslateDNA(g.Seq())
	return strings.TrimRight(prot, "*")
}

func (g Gene) GetCodon(pos int) (string, error) {
	prot := g.Prot()
	if pos > len(prot) {
		return "", fmt.Errorf("there are only %d aminoacids in this gene", len(prot))
	}
	seq := g.Seq()
	return seq[(3*(pos-1)) : pos*3], nil
}

func (g Gene) GetReferenceCodon(pos int) (string, error) {
	codon, err := g.GetCodon(pos)
	if err != nil {
		return "", err
	}
	if g.Forward {
		return codon, nil
	}
	return mykrobe.RevcompDNA(codon), nil
}

func (g Gene) GetReferenceCodons(pos int) ([]string, error) {
	refCodon, err := g.GetCodon(pos)
	if err != nil {
		return nil, err
	}
	refAA, ok := mykrobe.TranslateCodon(refCodon)
	if !ok {
		return nil, fmt.Errorf("invalid reference codon %q", refCodon)
	}
	codons := append([]string(nil), mykrobe.BackwardCodonTable()[refAA]...)
	if g.Forward {
		return codons, nil
	}
	for i := range codons {
		codons[i] = mykrobe.RevcompDNA(codons[i])
	}
	return codons, nil
}

type GeneAminoAcidChangeToDNAVariants struct {
	Reference         string
	Genes             map[string]Gene
	backwardCodonByAA map[string][]string
}

func NewGeneAminoAcidChangeToDNAVariants(referencePath, genbankPath string) (*GeneAminoAcidChangeToDNAVariants, error) {
	ref, err := loadReference(referencePath)
	if err != nil {
		return nil, err
	}
	genes, err := parseGenbankGenes(genbankPath, ref)
	if err != nil {
		return nil, err
	}
	return &GeneAminoAcidChangeToDNAVariants{
		Reference:         ref,
		Genes:             genes,
		backwardCodonByAA: mykrobe.BackwardCodonTable(),
	}, nil
}

func (g *GeneAminoAcidChangeToDNAVariants) GetAlts(aminoAcid string) []string {
	if aminoAcid == "X" {
		var out []string
		for aa, codons := range g.backwardCodonByAA {
			if aa == "*" {
				continue
			}
			out = append(out, codons...)
		}
		sort.Strings(out)
		return out
	}
	return append([]string(nil), g.backwardCodonByAA[aminoAcid]...)
}

func (g *GeneAminoAcidChangeToDNAVariants) GetReferenceAlts(gene Gene, aminoAcid string) []string {
	alts := g.GetAlts(aminoAcid)
	if gene.Forward {
		return alts
	}
	out := make([]string, len(alts))
	for i, alt := range alts {
		out[i] = mykrobe.RevcompDNA(alt)
	}
	return out
}

func (g *GeneAminoAcidChangeToDNAVariants) GetLocation(gene Gene, pos int) (int, error) {
	var dnaPos int
	if gene.Forward {
		dnaPos = (3 * (pos - 1)) + 1
	} else {
		dnaPos = 3 * pos
	}
	return gene.GetReferencePosition(dnaPos)
}

func (g *GeneAminoAcidChangeToDNAVariants) GetVariantNames(geneName, mutation string, proteinCodingVar bool) ([]string, error) {
	ref, start, alt, err := splitVarName(mutation)
	if err != nil {
		return nil, err
	}
	gene, err := g.GetGene(geneName)
	if err != nil {
		return nil, err
	}
	if start < 0 || !proteinCodingVar {
		return g.processDNAMutation(gene, ref, start, alt), nil
	}
	if start > 0 {
		return g.processCodingMutation(gene, ref, start, alt)
	}
	return nil, fmt.Errorf("variants are defined in 1-based coordinates. You can't have pos 0")
}

func (g *GeneAminoAcidChangeToDNAVariants) processDNAMutation(gene Gene, ref string, start int, alt string) []string {
	pos, _ := gene.GetReferencePosition(start)
	if !gene.Forward {
		pos -= len(ref) - 1
		ref = mykrobe.RevcompDNA(ref)
		if alt != "X" {
			alt = mykrobe.RevcompDNA(alt)
		}
	}
	if alt == "X" {
		var out []string
		for _, base := range []string{"A", "T", "C", "G"} {
			if base != ref {
				out = append(out, ref+itoa(pos)+base)
			}
		}
		return out
	}
	return []string{ref + itoa(pos) + alt}
}

func (g *GeneAminoAcidChangeToDNAVariants) processCodingMutation(gene Gene, ref string, start int, alt string) ([]string, error) {
	prot := gene.Prot()
	if prot == "" || start > len(prot) {
		return nil, fmt.Errorf("error translating %s_%s", gene.Name, ref+itoa(start)+alt)
	}
	if prot[start-1:start] != ref {
		return nil, fmt.Errorf("error processing %s_%s. The reference at pos %d is not %s, it's %s", gene.Name, ref+itoa(start)+alt, start, ref, prot[start-1:start])
	}
	refCodons, err := gene.GetReferenceCodons(start)
	if err != nil {
		return nil, err
	}
	altCodons := g.GetReferenceAlts(gene, alt)
	refSet := map[string]struct{}{}
	for _, c := range refCodons {
		refSet[c] = struct{}{}
	}
	filtered := altCodons[:0]
	for _, c := range altCodons {
		if _, ok := refSet[c]; ok {
			continue
		}
		filtered = append(filtered, c)
	}
	location, err := g.GetLocation(gene, start)
	if err != nil {
		return nil, err
	}
	refCodon, err := gene.GetReferenceCodon(start)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(filtered))
	for _, altCodon := range filtered {
		names = append(names, refCodon+itoa(location)+altCodon)
	}
	return names, nil
}

func (g *GeneAminoAcidChangeToDNAVariants) GetGene(name string) (Gene, error) {
	gene, ok := g.Genes[name]
	if !ok {
		return Gene{}, fmt.Errorf("gene %q not found", name)
	}
	return gene, nil
}

func loadReference(path string) (string, error) {
	reader, err := seqio.OpenPath(path)
	if err != nil {
		return "", err
	}
	defer closeIfPossible(reader)
	var seq strings.Builder
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		seq.WriteString(strings.ToUpper(string(rec.Seq)))
	}
	return seq.String(), nil
}

func parseGenbankGenes(path, ref string) (map[string]Gene, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	genes := map[string]Gene{}
	inFeatures := false
	var currentType, currentLocation, currentName string
	flush := func() error {
		if currentType != "gene" || currentName == "" || currentLocation == "" {
			return nil
		}
		m := locationRe.FindStringSubmatch(strings.TrimSpace(currentLocation))
		if m == nil {
			return nil
		}
		start, err := atoi(m[2])
		if err != nil {
			return err
		}
		end, err := atoi(m[3])
		if err != nil {
			return err
		}
		forward := m[1] == ""
		genes[currentName] = Gene{Name: currentName, Region: Region{Reference: ref, Start: start, End: end, Forward: forward}}
		return nil
	}
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "FEATURES"):
			inFeatures = true
		case strings.HasPrefix(line, "ORIGIN"):
			if err := flush(); err != nil {
				return nil, err
			}
			return genes, nil
		case !inFeatures:
			continue
		case len(line) >= 21 && strings.TrimSpace(line[:21]) != "":
			if err := flush(); err != nil {
				return nil, err
			}
			currentType = strings.TrimSpace(line[5:21])
			currentLocation = strings.TrimSpace(line[21:])
			currentName = ""
		case strings.HasPrefix(strings.TrimSpace(line), "/gene="):
			currentName = strings.Trim(strings.TrimPrefix(strings.TrimSpace(line), "/gene="), "\"")
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return genes, nil
}

func splitVarName(name string) (string, int, string, error) {
	m := splitVarRe.FindStringSubmatch(strings.ToUpper(name))
	if m == nil {
		return "", 0, "", fmt.Errorf("invalid mutation %q", name)
	}
	n, err := atoi(m[2])
	if err != nil {
		return "", 0, "", err
	}
	return m[1], n, m[3], nil
}

func atoi(s string) (int, error) {
	sign := 1
	if strings.HasPrefix(s, "-") {
		sign = -1
		s = s[1:]
	}
	if s == "" {
		return 0, fmt.Errorf("invalid integer")
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid integer %q", s)
		}
		n = n*10 + int(r-'0')
	}
	return sign * n, nil
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func closeIfPossible(v any) {
	if c, ok := v.(io.Closer); ok {
		_ = c.Close()
	}
}
