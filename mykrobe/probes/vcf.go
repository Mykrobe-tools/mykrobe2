package probes

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func LoadVCFMutations(path, reference string) ([]Mutation, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var mutations []Mutation
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "##") || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 8 {
			return nil, fmt.Errorf("expected at least 8 VCF columns in %s", path)
		}
		pos := fields[1]
		ref := strings.ToUpper(fields[3])
		filter := fields[6]
		if filter != "." && filter != "PASS" {
			continue
		}
		for _, alt := range strings.Split(fields[4], ",") {
			alt = strings.ToUpper(strings.TrimSpace(alt))
			if alt == "" || alt == "." || strings.HasPrefix(alt, "<") {
				continue
			}
			mutations = append(mutations, Mutation{
				Reference: reference,
				VarName:   ref + pos + alt,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return mutations, nil
}
