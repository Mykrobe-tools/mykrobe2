package probes

type Panel struct {
	Variant Variant
	Refs    []string
	Start   int
	Alts    []string
}

func newPanel(variant Variant, refs [][]byte, start int, alts [][]byte) Panel {
	refStrings := make([]string, 0, len(refs))
	for _, ref := range refs {
		refStrings = append(refStrings, string(ref))
	}
	altStrings := make([]string, 0, len(alts))
	for _, alt := range alts {
		altStrings = append(altStrings, string(alt))
	}
	refStrings = uniqueStrings(refStrings)
	altStrings = uniqueStrings(altStrings)
	refSet := make(map[string]struct{}, len(refStrings))
	for _, ref := range refStrings {
		refSet[ref] = struct{}{}
	}
	filteredAlts := altStrings[:0]
	for _, alt := range altStrings {
		if _, ok := refSet[alt]; ok {
			continue
		}
		filteredAlts = append(filteredAlts, alt)
	}
	refStrings = reorderRefsToAvoidOverlap(refStrings, filteredAlts)
	return Panel{Variant: variant, Refs: refStrings, Start: start, Alts: filteredAlts}
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func reorderRefsToAvoidOverlap(refs, alts []string) []string {
	if len(refs) != len(alts) || len(refs) <= 1 {
		return refs
	}
	alreadyGood := true
	for i := range refs {
		if hasOverlappingKmers(refs[i], alts[i], 31) {
			alreadyGood = false
			break
		}
	}
	if alreadyGood {
		return refs
	}
	edges := make([][]int, len(alts))
	for ai, alt := range alts {
		for ri, ref := range refs {
			if !hasOverlappingKmers(ref, alt, 31) {
				edges[ai] = append(edges[ai], ri)
			}
		}
		if len(edges[ai]) == 0 {
			return refs
		}
	}
	matchAltToRef := make([]int, len(alts))
	for i := range matchAltToRef {
		matchAltToRef[i] = -1
	}
	used := make([]bool, len(refs))
	var assign func(int) bool
	assign = func(ai int) bool {
		if ai == len(alts) {
			return true
		}
		for _, ri := range edges[ai] {
			if used[ri] {
				continue
			}
			used[ri] = true
			matchAltToRef[ai] = ri
			if assign(ai + 1) {
				return true
			}
			matchAltToRef[ai] = -1
			used[ri] = false
		}
		return false
	}
	if !assign(0) {
		return refs
	}
	out := make([]string, len(refs))
	for ai, ri := range matchAltToRef {
		out[ai] = refs[ri]
	}
	return out
}

func hasOverlappingKmers(ref, alt string, k int) bool {
	if len(ref) < k || len(alt) < k {
		return false
	}
	seen := make(map[string]struct{}, len(ref)-k+1)
	for i := 0; i <= len(ref)-k; i++ {
		seen[ref[i:i+k]] = struct{}{}
	}
	for i := 0; i <= len(alt)-k; i++ {
		if _, ok := seen[alt[i:i+k]]; ok {
			return true
		}
	}
	return false
}

func informativeKmerIndexes(seq string, k int, keep func(string) bool) []int {
	if len(seq) < k {
		return nil
	}
	var out []int
	for i := 0; i <= len(seq)-k; i++ {
		if keep(seq[i : i+k]) {
			out = append(out, i)
		}
	}
	return out
}
