package probes

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/martinghunt/mykrobe2/mykrobe/annotation"
)

type RunOptions struct {
	ReferencePath  string
	VCFPath        string
	BackgroundVCF  []string
	BackgroundList string
	Variants       []string
	TextFile       string
	GenbankPath    string
	Kmer           int
}

func WritePanels(w io.Writer, opts RunOptions) (map[string]LineageInfo, error) {
	if opts.VCFPath != "" && (opts.TextFile != "" || len(opts.Variants) > 0) {
		return nil, fmt.Errorf("make-probes --vcf cannot be combined with --variants or --text_file")
	}
	if opts.VCFPath == "" && len(opts.Variants) == 0 && opts.TextFile == "" {
		return nil, fmt.Errorf("make-probes requires --variants, --text_file, or --vcf")
	}

	reference := DefaultReferenceName(opts.ReferencePath)
	mutations, lineages, err := loadMutations(opts, reference)
	if err != nil {
		return nil, err
	}
	ag, err := NewAlleleGenerator(opts.ReferencePath, opts.Kmer)
	if err != nil {
		return nil, err
	}
	parsed, err := parseMutations(mutations)
	if err != nil {
		return nil, err
	}
	contextIndex, err := loadBackgroundContextIndex(opts, reference)
	if err != nil {
		return nil, err
	}

	for i, mut := range mutations {
		v := parsed[i]
		var context []Variant
		if contextIndex != nil {
			context = contextIndex.Nearby(v, -1, opts.Kmer)
		}
		panel, err := ag.Create(v, context)
		if err != nil {
			return nil, err
		}
		if err := writeProbePanel(w, mut, panel); err != nil {
			return nil, err
		}
	}
	return lineages, nil
}

func loadMutations(opts RunOptions, reference string) ([]Mutation, map[string]LineageInfo, error) {
	switch {
	case opts.VCFPath != "":
		mutations, err := LoadVCFMutations(opts.VCFPath, reference)
		return mutations, nil, err
	case opts.GenbankPath != "":
		mutations, err := loadGenbankMutations(opts, reference)
		return mutations, nil, err
	case opts.TextFile != "":
		return LoadDNAVarsTextFile(opts.TextFile, reference)
	default:
		mutations := make([]Mutation, 0, len(opts.Variants))
		for _, v := range opts.Variants {
			mutations = append(mutations, Mutation{Reference: reference, VarName: v})
		}
		return mutations, nil, nil
	}
}

func loadGenbankMutations(opts RunOptions, reference string) ([]Mutation, error) {
	aa2dna, err := annotation.NewGeneAminoAcidChangeToDNAVariants(opts.ReferencePath, opts.GenbankPath)
	if err != nil {
		return nil, err
	}
	if opts.TextFile != "" {
		rows, err := loadGenbankMutationRows(opts.TextFile)
		if err != nil {
			return nil, err
		}
		var mutations []Mutation
		for _, row := range rows {
			proteinCodingVar := row.alphabet != "DNA"
			varNames, err := aa2dna.GetVariantNames(row.gene, row.mutation, proteinCodingVar)
			if err != nil {
				return nil, err
			}
			for _, varName := range varNames {
				mutations = append(mutations, Mutation{
					Reference:         reference,
					VarName:           varName,
					InputMutationName: row.mutation,
					ProteinCodingVar:  proteinCodingVar,
				})
			}
		}
		return mutations, nil
	}

	mutations := make([]Mutation, 0, len(opts.Variants))
	for _, item := range opts.Variants {
		parts := strings.SplitN(item, "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("expected genbank variant in gene_mutation form, got %q", item)
		}
		varNames, err := aa2dna.GetVariantNames(parts[0], parts[1], true)
		if err != nil {
			return nil, err
		}
		for _, varName := range varNames {
			mutations = append(mutations, Mutation{
				Reference:         reference,
				VarName:           varName,
				InputMutationName: parts[1],
				ProteinCodingVar:  true,
			})
		}
	}
	return mutations, nil
}

func parseMutations(mutations []Mutation) ([]Variant, error) {
	parsed := make([]Variant, 0, len(mutations))
	for _, mut := range mutations {
		v, err := mut.Variant()
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, v)
	}
	return parsed, nil
}

func loadBackgroundContextIndex(opts RunOptions, reference string) (*ContextIndex, error) {
	backgroundPaths, err := collectBackgroundVCFPaths(opts.BackgroundVCF, opts.BackgroundList)
	if err != nil {
		return nil, err
	}
	if len(backgroundPaths) == 0 {
		return nil, nil
	}
	var backgroundVars []Variant
	for _, path := range backgroundPaths {
		bgMuts, err := LoadVCFMutations(path, reference)
		if err != nil {
			return nil, err
		}
		for _, mut := range bgMuts {
			v, err := mut.Variant()
			if err != nil {
				return nil, err
			}
			backgroundVars = append(backgroundVars, v)
		}
	}
	return NewContextIndex(backgroundVars), nil
}

func writeProbePanel(w io.Writer, mut Mutation, panel Panel) error {
	geneName := "NA"
	for i, ref := range panel.Refs {
		if _, err := fmt.Fprintf(w, ">ref-%s?var_name=%s&num_alts=%d&ref=%s&enum=%d&gene=%s&mut=%s\n%s\n",
			mut.MutationOutputName(), mut.VarName, len(panel.Alts), mut.Reference, i, geneName, mut.MutationOutputName(), ref); err != nil {
			return err
		}
	}
	for i, alt := range panel.Alts {
		if _, err := fmt.Fprintf(w, ">alt-%s?var_name=%s&enum=%d&gene=%s&mut=%s\n%s\n",
			mut.MutationOutputName(), mut.VarName, i, geneName, mut.MutationOutputName(), alt); err != nil {
			return err
		}
	}
	return nil
}

func collectBackgroundVCFPaths(direct []string, listFile string) ([]string, error) {
	out := append([]string(nil), direct...)
	if listFile == "" {
		return out, nil
	}
	f, err := os.Open(listFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

type genbankMutationRow struct {
	gene     string
	mutation string
	alphabet string
}

func loadGenbankMutationRows(path string) ([]genbankMutationRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows []genbankMutationRow
	r := csv.NewReader(f)
	r.Comma = '\t'
	r.FieldsPerRecord = -1
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(row) < 3 {
			return nil, fmt.Errorf("expected 3 columns in %s", path)
		}
		rows = append(rows, genbankMutationRow{gene: row[0], mutation: row[1], alphabet: row[2]})
	}
	return rows, nil
}
