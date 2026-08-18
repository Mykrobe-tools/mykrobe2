package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeProbesCommandWithVariants(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "probes.fa")
	oldStdout := os.Stdout
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	os.Stdout = f
	defer func() { os.Stdout = oldStdout }()

	err = run([]string{
		"make-probes",
		filepath.Join(mykrobeTestRefData, "BX571856.1.fasta"),
		"--variants", "A31T",
		"--kmer", "31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, ">ref-A31T?var_name=A31T") {
		t.Fatalf("missing ref record in output: %s", text)
	}
	if !strings.Contains(text, ">alt-A31T?var_name=A31T") {
		t.Fatalf("missing alt record in output: %s", text)
	}
}

func TestMakeProbesCommandWithTextFileWritesLineage(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "probes.fa")
	lineage := filepath.Join(dir, "lineage.json")
	refPath := filepath.Join(dir, "ref.fa")
	textFile := filepath.Join(dir, "vars.tsv")
	refSeq := []byte(strings.Repeat("A", 80))
	refSeq[41] = 'G'
	refSeq[51] = 'C'
	refSeq[61] = 'C'
	refSeq[71] = 'A'
	if err := os.WriteFile(refPath, []byte(">ref\n"+string(refSeq)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(textFile, []byte("ref\t42\tG\tA\tDNA\tlineage1\nref\t52\tC\tG\tDNA\t*lineage1.2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	os.Stdout = f
	defer func() { os.Stdout = oldStdout }()

	err = run([]string{
		"make-probes",
		refPath,
		"--text_file", textFile,
		"--lineage", lineage,
		"--kmer", "5",
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(lineage)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\"G42A\"") {
		t.Fatalf("missing lineage output: %s", string(data))
	}
}

func TestMakeProbesCommandWithGenbankVariant(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "probes.fa")
	oldStdout := os.Stdout
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	os.Stdout = f
	defer func() { os.Stdout = oldStdout }()

	err = run([]string{
		"make-probes",
		filepath.Join(mykrobeTestRefData, "NC_000962.3.fasta"),
		"--genbank", filepath.Join(mykrobeTestRefData, "NC_000962.3.gb"),
		"--variants", "rpoB_S450L",
		"--kmer", "31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, ">ref-S450L?var_name=TCG761154TTA") {
		t.Fatalf("missing expected genbank ref record: %s", text)
	}
	if !strings.Contains(text, ">alt-S450L?var_name=TCG761154TTA") {
		t.Fatalf("missing expected genbank alt record: %s", text)
	}
}

func TestMakeProbesCommandWithGenbankTextFileIncludesGeneName(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "probes.fa")
	textFile := filepath.Join(dir, "vars.tsv")
	if err := os.WriteFile(textFile, []byte("rpoB\tS450L\tPROT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	os.Stdout = f
	defer func() { os.Stdout = oldStdout }()

	err = run([]string{
		"make-probes",
		filepath.Join(mykrobeTestRefData, "NC_000962.3.fasta"),
		"--genbank", filepath.Join(mykrobeTestRefData, "NC_000962.3.gb"),
		"--text_file", textFile,
		"--kmer", "31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "&gene=rpoB&mut=S450L") {
		t.Fatalf("missing expected genbank gene metadata: %s", text)
	}
}

func TestMakeProbesCommandWithVCF(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "probes.fa")
	vcf := filepath.Join(dir, "vars.vcf")
	vcfData := "" +
		"##fileformat=VCFv4.2\n" +
		"#CHROM\tPOS\tID\tREF\tALT\tQUAL\tFILTER\tINFO\n" +
		"ref\t31\t.\tA\tT\t.\tPASS\t.\n" +
		"ref\t32\t.\tA\tG\t.\tPASS\t.\n"
	if err := os.WriteFile(vcf, []byte(vcfData), 0o644); err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	os.Stdout = f
	defer func() { os.Stdout = oldStdout }()

	err = run([]string{
		"make-probes",
		filepath.Join(mykrobeTestRefData, "BX571856.1.fasta"),
		"--vcf", vcf,
		"--kmer", "31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, ">ref-A31T?var_name=A31T") {
		t.Fatalf("missing first VCF record in output: %s", text)
	}
	if !strings.Contains(text, ">alt-A32G?var_name=A32G") {
		t.Fatalf("missing second VCF record in output: %s", text)
	}
}

func TestMakeProbesCommandUsesBackgroundVCFContext(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "probes.fa")
	bgVCF := filepath.Join(dir, "background.vcf")
	vcfData := "" +
		"##fileformat=VCFv4.2\n" +
		"#CHROM\tPOS\tID\tREF\tALT\tQUAL\tFILTER\tINFO\n" +
		"ref\t32\t.\tA\tT\t.\tPASS\t.\n"
	if err := os.WriteFile(bgVCF, []byte(vcfData), 0o644); err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	os.Stdout = f
	defer func() { os.Stdout = oldStdout }()

	err = run([]string{
		"make-probes",
		filepath.Join(mykrobeTestRefData, "BX571856.1.fasta"),
		"--variants", "A31T",
		"--background-vcf", bgVCF,
		"--kmer", "31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, ">ref-A31T?var_name=A31T&num_alts=2") {
		t.Fatalf("expected context-expanded A31T refs in output: %s", text)
	}
	if !strings.Contains(text, "CGATTAAAGATAGAAATACACGATGCGAGCATTCAAATTTCATAACATCACCATGAGTTTG") {
		t.Fatalf("expected background-specific reference sequence in output: %s", text)
	}
	if !strings.Contains(text, "CGATTAAAGATAGAAATACACGATGCGAGCTTTCAAATTTCATAACATCACCATGAGTTTG") {
		t.Fatalf("expected background-specific alternate sequence in output: %s", text)
	}
}

func TestMakeProbesCommandWithoutBackgroundInputsDoesNotExpandContext(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "probes.fa")
	oldStdout := os.Stdout
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	os.Stdout = f
	defer func() { os.Stdout = oldStdout }()

	err = run([]string{
		"make-probes",
		filepath.Join(mykrobeTestRefData, "BX571856.1.fasta"),
		"--variants", "A31T",
		"--kmer", "31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, ">ref-A31T?var_name=A31T&num_alts=1") {
		t.Fatalf("expected no-background A31T refs in output: %s", text)
	}
	if strings.Contains(text, "CGATTAAAGATAGAAATACACGATGCGAGCTTTCAAATTTCATAACATCACCATGAGTTTG") {
		t.Fatalf("unexpected background-expanded alternate without background inputs: %s", text)
	}
}

func TestMakeProbesCommandUsesBackgroundVCFList(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "probes.fa")
	bgVCF := filepath.Join(dir, "background.vcf")
	bgList := filepath.Join(dir, "backgrounds.txt")
	vcfData := "" +
		"##fileformat=VCFv4.2\n" +
		"#CHROM\tPOS\tID\tREF\tALT\tQUAL\tFILTER\tINFO\n" +
		"ref\t32\t.\tA\tT\t.\tPASS\t.\n"
	if err := os.WriteFile(bgVCF, []byte(vcfData), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bgList, []byte("# comment\n\n"+bgVCF+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	os.Stdout = f
	defer func() { os.Stdout = oldStdout }()

	err = run([]string{
		"make-probes",
		filepath.Join(mykrobeTestRefData, "BX571856.1.fasta"),
		"--variants", "A31T",
		"--background-vcf-list", bgList,
		"--kmer", "31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, ">ref-A31T?var_name=A31T&num_alts=2") {
		t.Fatalf("expected context-expanded A31T refs in output: %s", text)
	}
	if !strings.Contains(text, "CGATTAAAGATAGAAATACACGATGCGAGCTTTCAAATTTCATAACATCACCATGAGTTTG") {
		t.Fatalf("expected background-expanded alternate sequence in output: %s", text)
	}
}
