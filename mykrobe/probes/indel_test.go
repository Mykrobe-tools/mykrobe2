package probes

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestSimpleDeletion1(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "AA31A")
	if !v.IsIndel() || !v.IsDeletion() {
		t.Fatalf("expected deletion variant")
	}
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	if !slices.Contains(panel.Refs, "CGATTAAAGATAGAAATACACGATGCGAGCAATCAAATTTCATAACATCACCATGAGTTTG") {
		t.Fatalf("missing expected ref sequence: %v", panel.Refs)
	}
	wantAlts := []string{"GATTAAAGATAGAAATACACGATGCGAGCATCAAATTTCATAACATCACCATGAGTTTG"}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
	if ag.calculateLengthDeltaFromIndels(v, nil) != 1 {
		t.Fatalf("unexpected delta")
	}
}

func TestSimpleDeletion2(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "AT32A")
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	wantAlts := []string{"ATTAAAGATAGAAATACACGATGCGAGCAACAAATTTCATAACATCACCATGAGTTTGAT"}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestSimpleDeletion3(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "AT2902618T")
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	wantAlts := []string{"TTTATACTACTGCTCAATTTTTTTACTTTTTNNNNNNNNNNNNNNNNNNNNNNNNNNNNNN"}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestSimpleDeletion4(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "ATC32A")
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	wantAlts := []string{"ATTAAAGATAGAAATACACGATGCGAGCAAAAATTTCATAACATCACCATGAGTTTGAT"}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestSimpleInsertion1(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "C1TTTC")
	if !v.IsIndel() || !v.IsInsertion() {
		t.Fatalf("expected insertion variant")
	}
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantAlts := []string{"TTTCGATTAAAGATAGAAATACACGATGCGAGC"}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestSimpleInsertion2(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "C1CTTT")
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantAlts := []string{"CTTTGATTAAAGATAGAAATACACGATGCGAGCA"}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestSimpleInsertion3(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "A31ATTT")
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	wantAlts := []string{"GATTAAAGATAGAAATACACGATGCGAGCATTTATCAAATTTCATAACATCACCATGAGTTTG"}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestSimpleInsertion4(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "A32AGGGG")
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	wantAlts := []string{"ATTAAAGATAGAAATACACGATGCGAGCAAGGGGTCAAATTTCATAACATCACCATGAGTTTGA"}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestSimpleInsertion5(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "A2902618ATGC")
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	wantAlts := []string{"TATACTACTGCTCAATTTTTTTACTTTTATGCTNNNNNNNNNNNNNNNNNNNNNNNNNNNNN"}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestInsertionWithSNPContext(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "A31ATTT")
	v2 := mustVariant(t, "ref", "A32T")
	panel, err := ag.Create(v, []Variant{v2})
	if err != nil {
		t.Fatal(err)
	}
	wantAlts := []string{
		"GATTAAAGATAGAAATACACGATGCGAGCATTTATCAAATTTCATAACATCACCATGAGTTTG",
		"TTAAAGATAGAAATACACGATGCGAGCATTTTTCAAATTTCATAACATCACCATGAGTTTG",
	}
	slices.Sort(panel.Alts)
	slices.Sort(wantAlts)
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestDeletionWithSNPContext1(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "AA31A")
	v2 := mustVariant(t, "ref", "T33A")
	panel, err := ag.Create(v, []Variant{v2})
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	wantAlts := []string{
		"ATTAAAGATAGAAATACACGATGCGAGCAACAAATTTCATAACATCACCATGAGTTTGA",
		"GATTAAAGATAGAAATACACGATGCGAGCATCAAATTTCATAACATCACCATGAGTTTG",
	}
	slices.Sort(panel.Alts)
	slices.Sort(wantAlts)
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestDeletionWithSNPContext2(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	v := mustVariant(t, "ref", "AA31A")
	v2 := mustVariant(t, "ref", "A32T")
	got := ag.removeOverlappingContexts(v, []Variant{v2})
	if len(got) != 0 {
		t.Fatalf("expected overlapping context to be removed, got %v", got)
	}
	panel, err := ag.Create(v, []Variant{v2})
	if err != nil {
		t.Fatal(err)
	}
	wantAlts := []string{"GATTAAAGATAGAAATACACGATGCGAGCATCAAATTTCATAACATCACCATGAGTTTG"}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestLargeVariant1(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "NC_000962.3.fasta"), 31)
	v := mustVariant(t, "ref", "AACGCCCGGTATCTGAGGATCTGTGTTCTCACCCAATACAAGTCGCATTCACT1355983ACCGCCCGGTATCTGAGGATTGGTTTTCCACCCAAATACAAGTCGCATTCGCG")
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	wantAlts := []string{"TCGTCACCGCCCGGTATCTGAGGATTGGTTTTCCACCCAAATACAAGTCGCATTCGCGGGA"}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestLargeInsertionVariant(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "NC_000962.3.fasta"), 31)
	v := mustVariant(t, "ref", "C2352065CCTCGCCTGGGCTGGCGAGCAGACGCAAAATCCCCCGCACGCCCGGCGTGTCGGGGGATTTTGCGTCTG")
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	wantAlts := []string{"CCAGCTCAGTCACGTCGCCGCCGCCTCGCCTGGGCTGGCGAGCAGACGCAAAATCCCCCGCACGCCCGGCGTGTCGGGGGATTTTGCGTCTGCTCGCCAGTTGACCGCGCCCGCTCGCGGCT"}
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestSNPWithReplaceContext(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "NC_000962.3.fasta"), 31)
	v := mustVariant(t, "ref", "G2338961A")
	v1 := mustVariant(t, "ref", "GGATG2338990CGATA")
	panel, err := ag.Create(v, []Variant{v1})
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	if !slices.Contains(panel.Refs, "CGACTAGCCACCATCGCGCATCAGTGCGAGGTCAAAAGCGACCAAAGCGAGCAAGTCGCGG") {
		t.Fatalf("missing expected ref sequence: %v", panel.Refs)
	}
	wantAlts := []string{
		"CGACTAGCCACCATCGCGCATCAGTGCGAGATCAAAAGCGACCAAAGCGAGCAAGTCGCCG",
		"CGACTAGCCACCATCGCGCATCAGTGCGAGATCAAAAGCGACCAAAGCGAGCAAGTCGCGG",
	}
	slices.Sort(panel.Alts)
	slices.Sort(wantAlts)
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestIndelSNPIndelContext(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "NC_000962.3.fasta"), 31)
	v := mustVariant(t, "ref", "TCGCGTGGC4021459GCGAGCAGA")
	v1 := mustVariant(t, "ref", "A4021455ATCTAGCCGCAAG")
	v2 := mustVariant(t, "ref", "T4021489G")
	panel, err := ag.Create(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	if !slices.Contains(panel.Refs, "ATCATGCGATTCTGCGTCTGCTCGCGAGGCTCGCGTGGCCGCCGGCGCTGGCGGGCGATCT") {
		t.Fatalf("missing expected ref sequence: %v", panel.Refs)
	}
	panel, err = ag.Create(v, []Variant{v1, v2})
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	wantAlts := []string{
		"ATCATGCGATTCTGCGTCTGCTCGCGAGGCGCGAGCAGACGCCGGCGCTGGCGGGCGATCG",
		"ATCATGCGATTCTGCGTCTGCTCGCGAGGCGCGAGCAGACGCCGGCGCTGGCGGGCGATCT",
		"TGCGTCTGCTCGCGATCTAGCCGCAAGGGCGCGAGCAGACGCCGGCGCTGGCGGGCGATCG",
		"TGCGTCTGCTCGCGATCTAGCCGCAAGGGCGCGAGCAGACGCCGGCGCTGGCGGGCGATCT",
	}
	slices.Sort(panel.Alts)
	slices.Sort(wantAlts)
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}

func TestComplexContext(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "NC_000962.3.fasta"), 31)
	v := mustVariant(t, "ref", "ATTT1503643A")
	v1 := mustVariant(t, "ref", "CCT1503615C")
	v2 := mustVariant(t, "ref", "A1503655ATGCCGCCGCC")
	panel, err := ag.Create(v, []Variant{v1, v2})
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlappingKmers(t, panel, 31)
	if !slices.Contains(panel.Refs, "ATCCTGGAGCCCACCAGCGGAAACACCGGCATTTCGCTGGCGATGGCGGCCCGGTTGAAGG") {
		t.Fatalf("missing expected ref sequence: %v", panel.Refs)
	}
	wantAlts := []string{
		"CCATCGGAGCCCACCAGCGGAAACACCGGCACGCTGGCGATGGCGGCCCGGTTGAAGGGGT",
		"TCCTGGAGCCCACCAGCGGAAACACCGGCACGCTGGCGATGGCGGCCCGGTTGAAGGGG",
		"ATCGGAGCCCACCAGCGGAAACACCGGCACGCTGGCGATGCCGCCGCCTGGCGGCCCGG",
		"TCCTGGAGCCCACCAGCGGAAACACCGGCACGCTGGCGATGCCGCCGCCTGGCGGCCCGG",
	}
	slices.Sort(panel.Alts)
	slices.Sort(wantAlts)
	if !slices.Equal(panel.Alts, wantAlts) {
		t.Fatalf("got %v want %v", panel.Alts, wantAlts)
	}
}
