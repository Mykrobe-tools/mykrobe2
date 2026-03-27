package probes

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadVCFMutations(t *testing.T) {
	dir := t.TempDir()
	vcf := filepath.Join(dir, "vars.vcf")
	data := "" +
		"##fileformat=VCFv4.2\n" +
		"#CHROM\tPOS\tID\tREF\tALT\tQUAL\tFILTER\tINFO\n" +
		"ref\t31\t.\tA\tT\t.\tPASS\t.\n" +
		"ref\t32\t.\tA\tT,G\t.\t.\t.\n" +
		"ref\t40\t.\tC\tA\t.\tLowQual\t.\n" +
		"ref\t41\t.\tG\t<DEL>\t.\tPASS\t.\n"
	if err := os.WriteFile(vcf, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadVCFMutations(vcf, "ref")
	if err != nil {
		t.Fatal(err)
	}
	want := []Mutation{
		{Reference: "ref", VarName: "A31T"},
		{Reference: "ref", VarName: "A32T"},
		{Reference: "ref", VarName: "A32G"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
