package probes

type ContextIndex struct {
	byStart map[int][]int
	vars    []Variant
}

func NewContextIndex(vars []Variant) *ContextIndex {
	idx := &ContextIndex{
		byStart: make(map[int][]int, len(vars)),
		vars:    append([]Variant(nil), vars...),
	}
	for i, v := range vars {
		idx.byStart[v.Start] = append(idx.byStart[v.Start], i)
	}
	return idx
}

func (c *ContextIndex) Nearby(target Variant, excludeIndex int, k int) []Variant {
	if c == nil {
		return nil
	}
	out := make([]Variant, 0)
	seen := make(map[int]struct{})
	for pos := target.Start - k; pos <= target.Start+k; pos++ {
		for _, idx := range c.byStart[pos] {
			if idx == excludeIndex {
				continue
			}
			if _, ok := seen[idx]; ok {
				continue
			}
			seen[idx] = struct{}{}
			out = append(out, c.vars[idx])
		}
	}
	return out
}
