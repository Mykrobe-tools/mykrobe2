package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/martinghunt/mykrobe2/mykrobe/annotation"
	"github.com/martinghunt/mykrobe2/mykrobe/probes"
)

func runMakeProbes(opts *makeProbesOptions, out io.Writer) error {
	if opts.vcfPath != "" && (opts.textFile != "" || len(opts.variants) > 0) {
		return fmt.Errorf("make-probes --vcf cannot be combined with --variants or --text_file")
	}
	if opts.vcfPath == "" && len(opts.variants) == 0 && opts.textFile == "" {
		return fmt.Errorf("make-probes requires --variants, --text_file, or --vcf")
	}

	reference := probes.DefaultReferenceName(opts.referencePath)
	mutations, err := loadProbeMutations(opts, reference)
	if err != nil {
		return err
	}
	ag, err := probes.NewAlleleGenerator(opts.referencePath, opts.kmer)
	if err != nil {
		return err
	}
	parsed, err := parseMutations(mutations)
	if err != nil {
		return err
	}
	contextIndex, err := loadBackgroundContextIndex(opts, reference)
	if err != nil {
		return err
	}

	for i, mut := range mutations {
		v := parsed[i]
		var context []probes.Variant
		if contextIndex != nil {
			context = contextIndex.Nearby(v, -1, opts.kmer)
		}
		panel, err := ag.Create(v, context)
		if err != nil {
			return err
		}
		if err := writeProbePanel(out, mut, panel); err != nil {
			return err
		}
	}
	return nil
}

func loadProbeMutations(opts *makeProbesOptions, reference string) ([]probes.Mutation, error) {
	switch {
	case opts.vcfPath != "":
		return probes.LoadVCFMutations(opts.vcfPath, reference)
	case opts.genbankPath != "":
		return loadGenbankMutations(opts, reference)
	case opts.textFile != "":
		mutations, lineages, err := probes.LoadDNAVarsTextFile(opts.textFile, reference)
		if err != nil {
			return nil, err
		}
		if err := writeLineages(opts.lineagePath, lineages); err != nil {
			return nil, err
		}
		return mutations, nil
	default:
		mutations := make([]probes.Mutation, 0, len(opts.variants))
		for _, v := range opts.variants {
			mutations = append(mutations, probes.Mutation{Reference: reference, VarName: v})
		}
		return mutations, nil
	}
}

func loadGenbankMutations(opts *makeProbesOptions, reference string) ([]probes.Mutation, error) {
	aa2dna, err := annotation.NewGeneAminoAcidChangeToDNAVariants(opts.referencePath, opts.genbankPath)
	if err != nil {
		return nil, err
	}
	if opts.textFile != "" {
		rows, err := loadGenbankMutationRows(opts.textFile)
		if err != nil {
			return nil, err
		}
		var mutations []probes.Mutation
		for _, row := range rows {
			proteinCodingVar := row.alphabet != "DNA"
			varNames, err := aa2dna.GetVariantNames(row.gene, row.mutation, proteinCodingVar)
			if err != nil {
				return nil, err
			}
			for _, varName := range varNames {
				mutations = append(mutations, probes.Mutation{
					Reference:         reference,
					VarName:           varName,
					InputMutationName: row.mutation,
					ProteinCodingVar:  proteinCodingVar,
				})
			}
		}
		return mutations, nil
	}

	mutations := make([]probes.Mutation, 0, len(opts.variants))
	for _, item := range opts.variants {
		parts := strings.SplitN(item, "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("expected genbank variant in gene_mutation form, got %q", item)
		}
		varNames, err := aa2dna.GetVariantNames(parts[0], parts[1], true)
		if err != nil {
			return nil, err
		}
		for _, varName := range varNames {
			mutations = append(mutations, probes.Mutation{
				Reference:         reference,
				VarName:           varName,
				InputMutationName: parts[1],
				ProteinCodingVar:  true,
			})
		}
	}
	return mutations, nil
}

func writeLineages(path string, lineages map[string]probes.LineageInfo) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(lineages, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func parseMutations(mutations []probes.Mutation) ([]probes.Variant, error) {
	parsed := make([]probes.Variant, 0, len(mutations))
	for _, mut := range mutations {
		v, err := mut.Variant()
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, v)
	}
	return parsed, nil
}

func loadBackgroundContextIndex(opts *makeProbesOptions, reference string) (*probes.ContextIndex, error) {
	backgroundPaths, err := collectBackgroundVCFPaths(opts.backgroundVCF, opts.backgroundList)
	if err != nil {
		return nil, err
	}
	if len(backgroundPaths) == 0 {
		return nil, nil
	}
	var backgroundVars []probes.Variant
	for _, path := range backgroundPaths {
		bgMuts, err := probes.LoadVCFMutations(path, reference)
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
	return probes.NewContextIndex(backgroundVars), nil
}

func writeProbePanel(out io.Writer, mut probes.Mutation, panel probes.Panel) error {
	geneName := "NA"
	for i, ref := range panel.Refs {
		if _, err := fmt.Fprintf(out, ">ref-%s?var_name=%s&num_alts=%d&ref=%s&enum=%d&gene=%s&mut=%s\n%s\n",
			mut.MutationOutputName(), mut.VarName, len(panel.Alts), mut.Reference, i, geneName, mut.MutationOutputName(), ref); err != nil {
			return err
		}
	}
	for i, alt := range panel.Alts {
		if _, err := fmt.Fprintf(out, ">alt-%s?var_name=%s&enum=%d&gene=%s&mut=%s\n%s\n",
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
