package mykrobe

import "sort"

func RevcompDNA(seq string) string {
	out := make([]byte, len(seq))
	for i := range seq {
		out[len(seq)-1-i] = ComplementBase(seq[i])
	}
	return string(out)
}

func ComplementBase(b byte) byte {
	switch b {
	case 'A':
		return 'T'
	case 'C':
		return 'G'
	case 'G':
		return 'C'
	case 'T':
		return 'A'
	default:
		return 'N'
	}
}

func TranslateCodon(codon string) (string, bool) {
	aa, ok := codonTable[codon]
	return aa, ok
}

func TranslateDNA(seq string) string {
	out := make([]byte, 0, len(seq)/3)
	for i := 0; i+3 <= len(seq); i += 3 {
		if aa, ok := codonTable[seq[i:i+3]]; ok {
			out = append(out, aa[0])
		} else {
			out = append(out, 'X')
		}
	}
	return string(out)
}

func BackwardCodonTable() map[string][]string {
	table := map[string][]string{}
	for codon, aa := range codonTable {
		table[aa] = append(table[aa], codon)
	}
	for aa := range table {
		sort.Strings(table[aa])
	}
	return table
}

var codonTable = map[string]string{
	"TTT": "F", "TTC": "F", "TTA": "L", "TTG": "L",
	"TCT": "S", "TCC": "S", "TCA": "S", "TCG": "S",
	"TAT": "Y", "TAC": "Y", "TAA": "*", "TAG": "*",
	"TGT": "C", "TGC": "C", "TGA": "*", "TGG": "W",
	"CTT": "L", "CTC": "L", "CTA": "L", "CTG": "L",
	"CCT": "P", "CCC": "P", "CCA": "P", "CCG": "P",
	"CAT": "H", "CAC": "H", "CAA": "Q", "CAG": "Q",
	"CGT": "R", "CGC": "R", "CGA": "R", "CGG": "R",
	"ATT": "I", "ATC": "I", "ATA": "I", "ATG": "M",
	"ACT": "T", "ACC": "T", "ACA": "T", "ACG": "T",
	"AAT": "N", "AAC": "N", "AAA": "K", "AAG": "K",
	"AGT": "S", "AGC": "S", "AGA": "R", "AGG": "R",
	"GTT": "V", "GTC": "V", "GTA": "V", "GTG": "V",
	"GCT": "A", "GCC": "A", "GCA": "A", "GCG": "A",
	"GAT": "D", "GAC": "D", "GAA": "E", "GAG": "E",
	"GGT": "G", "GGC": "G", "GGA": "G", "GGG": "G",
}
