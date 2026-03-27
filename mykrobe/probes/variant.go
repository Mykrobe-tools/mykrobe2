package probes

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
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

func sameVariant(a, b Variant) bool {
	if a.Reference != b.Reference || a.Start != b.Start || a.ReferenceBases != b.ReferenceBases {
		return false
	}
	return slices.Equal(a.AlternateBases, b.AlternateBases)
}

func DefaultReferenceName(path string) string {
	return strings.Split(filepath.Base(path), ".fa")[0]
}
