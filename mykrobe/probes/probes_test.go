package probes

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/martinghunt/mykrobe2/internal/testutil"
)

var probeTestRefData = testutil.MykrobePath("tests", "ref_data")

func TestLoadDNAVarsTextFile(t *testing.T) {
	infile := testutil.MykrobePath("tests", "probe_tests", "test_probe_generation.load_dna_vars_txt_file.tsv")
	gotMutations, gotLineage, err := LoadDNAVarsTextFile(infile, "ref")
	if err != nil {
		t.Fatal(err)
	}
	wantMutations := []Mutation{
		{Reference: "ref", VarName: "G42A"},
		{Reference: "ref", VarName: "C52G"},
		{Reference: "ref", VarName: "C62G"},
		{Reference: "ref", VarName: "A72T"},
	}
	wantLineage := map[string]LineageInfo{
		"G42A": {Name: "lineage1", UseRefAllele: false},
		"C52G": {Name: "lineage1.2", UseRefAllele: true},
		"A72T": {Name: "lineage1.2.3", UseRefAllele: false, ReportName: "lineage1.2.3_report_name"},
	}
	if !slices.Equal(gotMutations, wantMutations) {
		t.Fatalf("mutations mismatch\ngot  %#v\nwant %#v", gotMutations, wantMutations)
	}
	if len(gotLineage) != len(wantLineage) {
		t.Fatalf("lineage size mismatch got=%v want=%v", gotLineage, wantLineage)
	}
	for k, want := range wantLineage {
		if gotLineage[k] != want {
			t.Fatalf("lineage mismatch for %s\ngot  %#v\nwant %#v", k, gotLineage[k], want)
		}
	}
}

func TestSimpleSNPVariant(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "A31T")
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	wantRefs := []string{"CGATTAAAGATAGAAATACACGATGCGAGCAATCAAATTTCATAACATCACCATGAGTTTG"}
	wantAlts := []string{"CGATTAAAGATAGAAATACACGATGCGAGCTATCAAATTTCATAACATCACCATGAGTTTG"}
	if !slices.Equal(panel.Refs, wantRefs) {
		t.Fatalf("refs mismatch\ngot  %v\nwant %v", panel.Refs, wantRefs)
	}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("alts mismatch\ngot  %v\nwant %v", panel.Alts, wantAlts)
	}
	if ag.calculateLengthDeltaFromIndels(v, nil) != 0 {
		t.Fatalf("unexpected length delta")
	}
	if v.IsIndel() {
		t.Fatalf("expected SNP, got indel")
	}
}

func TestSimpleVariantStart(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "C1T")
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantRefs := []string{"CGATTAAAGATAGAAATACACGATGCGAGCAATCAAATTTCATAACATCACCATGAGTTTG"}
	wantAlts := []string{"TGATTAAAGATAGAAATACACGATGCGAGCAATCAAATTTCATAACATCACCATGAGTTTG"}
	if !slices.Equal(panel.Refs, wantRefs) || !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("unexpected panel refs=%v alts=%v", panel.Refs, panel.Alts)
	}
}

func TestSimpleVariantEnd(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "A2902618T")
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantRefs := []string{"TTTATACTACTGCTCAATTTTTTTACTTTTATNNNNNNNNNNNNNNNNNNNNNNNNNNNNN"}
	wantAlts := []string{"TTTATACTACTGCTCAATTTTTTTACTTTTTTNNNNNNNNNNNNNNNNNNNNNNNNNNNNN"}
	if !slices.Equal(panel.Refs, wantRefs) || !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("unexpected panel refs=%v alts=%v", panel.Refs, panel.Alts)
	}
	assertNoOverlappingKmers(t, panel, 31)
}

func TestSimpleVariantWithNearbySNP(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "A31T")
	v2 := mustVariant(t, "ref", "A32T")
	panel, err := ag.Create(v, []Variant{v2})
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	wantRefs := []string{
		"CGATTAAAGATAGAAATACACGATGCGAGCAATCAAATTTCATAACATCACCATGAGTTTG",
		"CGATTAAAGATAGAAATACACGATGCGAGCATTCAAATTTCATAACATCACCATGAGTTTG",
	}
	wantAlts := []string{
		"CGATTAAAGATAGAAATACACGATGCGAGCTATCAAATTTCATAACATCACCATGAGTTTG",
		"CGATTAAAGATAGAAATACACGATGCGAGCTTTCAAATTTCATAACATCACCATGAGTTTG",
	}
	slices.Sort(panel.Refs)
	slices.Sort(panel.Alts)
	slices.Sort(wantRefs)
	slices.Sort(wantAlts)
	if !slices.Equal(panel.Refs, wantRefs) || !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("unexpected panel refs=%v alts=%v", panel.Refs, panel.Alts)
	}
}

func TestSimpleVariantWithMultipleNearbySNPs(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "A31T")
	v2 := mustVariant(t, "ref", "A32T")
	v3 := mustVariant(t, "ref", "C30G")
	panel, err := ag.Create(v, []Variant{v2, v3})
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	wantRefs := []string{
		"CGATTAAAGATAGAAATACACGATGCGAGCAATCAAATTTCATAACATCACCATGAGTTTG",
		"CGATTAAAGATAGAAATACACGATGCGAGCATTCAAATTTCATAACATCACCATGAGTTTG",
		"CGATTAAAGATAGAAATACACGATGCGAGGAATCAAATTTCATAACATCACCATGAGTTTG",
		"CGATTAAAGATAGAAATACACGATGCGAGGATTCAAATTTCATAACATCACCATGAGTTTG",
	}
	wantAlts := []string{
		"CGATTAAAGATAGAAATACACGATGCGAGCTATCAAATTTCATAACATCACCATGAGTTTG",
		"CGATTAAAGATAGAAATACACGATGCGAGCTTTCAAATTTCATAACATCACCATGAGTTTG",
		"CGATTAAAGATAGAAATACACGATGCGAGGTATCAAATTTCATAACATCACCATGAGTTTG",
		"CGATTAAAGATAGAAATACACGATGCGAGGTTTCAAATTTCATAACATCACCATGAGTTTG",
	}
	if !slices.Equal(panel.Refs, wantRefs) || !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("unexpected panel refs=%v alts=%v", panel.Refs, panel.Alts)
	}
}

func TestThreeBaseSubstitutionTrimsLikePythonIndelLogic(t *testing.T) {
	ag := mustAlleleGenerator(t, writeAE005674Fixture(t), 21)
	v := mustVariant(t, "AE005674", "CGA2354611CAA")
	context := []Variant{
		mustVariant(t, "AE005674", "T2354600C"),
		mustVariant(t, "AE005674", "C2354601A"),
		mustVariant(t, "AE005674", "C2354601T"),
	}
	panel, err := ag.Create(v, context)
	if err != nil {
		t.Fatal(err)
	}
	wantAlts := []string{
		"CGATCGTGTCATAAACCGCCAAGTCACCATGGGGATGGTA",
		"CGATCGTGCCATAAACCGCCAAGTCACCATGGGGATGGTA",
		"CGATCGTGTTATAAACCGCCAAGTCACCATGGGGATGGTA",
		"CGATCGTGCTATAAACCGCCAAGTCACCATGGGGATGGTA",
		"CGATCGTGTAATAAACCGCCAAGTCACCATGGGGATGGTA",
		"CGATCGTGCAATAAACCGCCAAGTCACCATGGGGATGGTA",
	}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("unexpected trimmed alts\ngot  %v\nwant %v", panel.Alts, wantAlts)
	}
}

func writeAE005674Fixture(t *testing.T) string {
	t.Helper()
	const segmentStart = 2354590
	const trimmedAlt = "CGATCGTGTCATAAACCGCCAAGTCACCATGGGGATGGTA"
	ref := []byte(strings.Repeat("N", segmentStart+1+len(trimmedAlt)))
	copy(ref[segmentStart:], "A"+trimmedAlt)
	copy(ref[segmentStart+20:], "CGA")
	path := filepath.Join(t.TempDir(), "AE005674.fasta")
	if err := os.WriteFile(path, append([]byte(">AE005674\n"), ref...), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSplitContext(t *testing.T) {
	v := mustVariant(t, "ref", "A31T")
	v3 := mustVariant(t, "ref", "C30G")
	v4 := mustVariant(t, "ref", "C30T")
	got := splitContext([]Variant{v, v3, v4})
	want := [][]Variant{{v, v4}, {v, v3}}
	if !sameVariantSets(got, want) {
		t.Fatalf("split context mismatch got=%v want=%v", got, want)
	}
}

func mustAlleleGenerator(t *testing.T, refPath string, k int) *AlleleGenerator {
	t.Helper()
	ag, err := NewAlleleGenerator(refPath, k)
	if err != nil {
		t.Fatal(err)
	}
	return ag
}

func mustVariant(t *testing.T, reference, name string) Variant {
	t.Helper()
	v, err := ParseVariantName(reference, name)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func assertNoOverlappingKmers(t *testing.T, panel Panel, k int) {
	t.Helper()
	for i := range panel.Refs {
		refKmers := seqToKmers(panel.Refs[i], k)
		altKmers := seqToKmers(panel.Alts[i], k)
		for _, ref := range refKmers {
			if slices.Contains(altKmers, ref) {
				t.Fatalf("found overlapping kmer %q in ref=%q alt=%q", ref, panel.Refs[i], panel.Alts[i])
			}
		}
	}
}

func seqToKmers(seq string, k int) []string {
	if len(seq) < k {
		return nil
	}
	out := make([]string, 0, len(seq)-k+1)
	for i := 0; i <= len(seq)-k; i++ {
		out = append(out, seq[i:i+k])
	}
	return out
}

func sameVariantSets(got, want [][]Variant) bool {
	if len(got) != len(want) {
		return false
	}
	norm := func(vs [][]Variant) []string {
		out := make([]string, 0, len(vs))
		for _, set := range vs {
			names := make([]string, 0, len(set))
			for _, v := range set {
				names = append(names, v.ReferenceBases+itoa(v.Start)+v.AlternateBases[0])
			}
			slices.Sort(names)
			out = append(out, strings.Join(names, ","))
		}
		slices.Sort(out)
		return out
	}
	return slices.Equal(norm(got), norm(want))
}
