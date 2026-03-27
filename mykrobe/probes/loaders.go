package probes

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

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
