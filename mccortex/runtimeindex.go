package mccortex

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"unsafe"

	"github.com/martinghunt/faqt/seqio"
)

const runtimeIndexMagic = "MYKROBE2RT2"

const invalidPanelKmer = ^uint64(0)

type RuntimeIndex struct {
	K            int
	TableKeys    []uint64
	ProbeOffsets []uint32
	ProbeSlots   []uint32
	NameOffsets  []uint32
	NameBytes    []byte
	KmerLengths  []uint32

	mappedData []byte
}

type runtimeBuildCounts struct {
	probeCount      int
	totalSlots      int
	totalNameBytes  int
	totalValidKmers int
}

func collectRuntimeBuildCounts(k int, paths []string) (runtimeBuildCounts, error) {
	var counts runtimeBuildCounts
	if k < 1 || k > 31 {
		return counts, fmt.Errorf("k must be in range 1..31, got %d", k)
	}
	for _, path := range paths {
		reader, err := seqio.OpenPath(path)
		if err != nil {
			return counts, err
		}
		for {
			rec, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				closeIfPossible(reader)
				return counts, err
			}
			counts.probeCount++
			counts.totalNameBytes += len(rec.Name)
			klen := 0
			if len(rec.Seq) >= k {
				klen = len(rec.Seq) - k + 1
			}
			counts.totalSlots += klen
			if klen <= 1 {
				continue
			}
			for i := 1; i < klen; i++ {
				_, ok, err := canonicalKmer(rec.Seq[i : i+k])
				if err != nil {
					closeIfPossible(reader)
					return counts, err
				}
				if !ok {
					continue
				}
				counts.totalValidKmers++
			}
		}
		closeIfPossible(reader)
	}
	return counts, nil
}

func buildRuntimeTableFromPaths(k int, paths []string, totalValidKmers int) ([]uint64, uint64, error) {
	tableKeys, mask := buildRuntimeTableSized(totalValidKmers)
	for _, path := range paths {
		reader, err := seqio.OpenPath(path)
		if err != nil {
			return nil, 0, err
		}
		for {
			rec, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				closeIfPossible(reader)
				return nil, 0, err
			}
			klen := 0
			if len(rec.Seq) >= k {
				klen = len(rec.Seq) - k + 1
			}
			if klen <= 1 {
				continue
			}
			for i := 1; i < klen; i++ {
				key, ok, err := canonicalKmer(rec.Seq[i : i+k])
				if err != nil {
					closeIfPossible(reader)
					return nil, 0, err
				}
				if !ok {
					continue
				}
				slot := lookupRuntimeSlot(tableKeys, mask, key)
				tableKeys[slot] = key
			}
		}
		closeIfPossible(reader)
	}
	return tableKeys, mask, nil
}

func BuildRuntimeIndex(k int, paths []string) (*RuntimeIndex, error) {
	counts, err := collectRuntimeBuildCounts(k, paths)
	if err != nil {
		return nil, err
	}
	tableKeys, mask, err := buildRuntimeTableFromPaths(k, paths, counts.totalValidKmers)
	if err != nil {
		return nil, err
	}
	rt := &RuntimeIndex{
		K:            k,
		TableKeys:    tableKeys,
		ProbeOffsets: make([]uint32, 0, counts.probeCount+1),
		ProbeSlots:   make([]uint32, 0, counts.totalSlots),
		NameOffsets:  make([]uint32, 0, counts.probeCount+1),
		NameBytes:    make([]byte, 0, counts.totalNameBytes),
		KmerLengths:  make([]uint32, 0, counts.probeCount),
	}
	rt.ProbeOffsets = append(rt.ProbeOffsets, 0)
	rt.NameOffsets = append(rt.NameOffsets, 0)
	for _, path := range paths {
		reader, err := seqio.OpenPath(path)
		if err != nil {
			return nil, err
		}
		for {
			rec, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				closeIfPossible(reader)
				return nil, err
			}
			klen := 0
			if len(rec.Seq) >= k {
				klen = len(rec.Seq) - k + 1
			}
			rt.KmerLengths = append(rt.KmerLengths, uint32(klen))
			rt.NameBytes = append(rt.NameBytes, rec.Name...)
			rt.NameOffsets = append(rt.NameOffsets, uint32(len(rt.NameBytes)))
			if klen <= 1 {
				rt.ProbeOffsets = append(rt.ProbeOffsets, uint32(len(rt.ProbeSlots)))
				continue
			}
			rt.ProbeSlots = append(rt.ProbeSlots, ^uint32(0))
			for i := 1; i < klen; i++ {
				key, ok, err := canonicalKmer(rec.Seq[i : i+k])
				if err != nil {
					closeIfPossible(reader)
					return nil, err
				}
				if !ok {
					rt.ProbeSlots = append(rt.ProbeSlots, ^uint32(0))
					continue
				}
				rt.ProbeSlots = append(rt.ProbeSlots, lookupRuntimeSlot(tableKeys, mask, key))
			}
			rt.ProbeOffsets = append(rt.ProbeOffsets, uint32(len(rt.ProbeSlots)))
		}
		closeIfPossible(reader)
	}
	return rt, nil
}

func BuildRuntimeIndexFile(path string, k int, paths []string) error {
	counts, err := collectRuntimeBuildCounts(k, paths)
	if err != nil {
		return err
	}
	tmpDir := filepath.Dir(path)
	size := runtimeTableSize(counts.totalValidKmers)
	var (
		tableFile   *os.File
		tablePath   string
		tableKeys   []uint64
		unmapTable  func() error
		tableMapped bool
		mask        uint64
	)
	if size > 0 {
		tableFile, tablePath, err = createRuntimeTableFile(tmpDir, size)
		if err != nil {
			return err
		}
		defer tableFile.Close()
		defer os.Remove(tablePath)
		tableKeys, _, unmapTable, err = mapUint64FileRW(tablePath, size)
		if err != nil {
			return err
		}
		tableMapped = true
		defer func() {
			if tableMapped {
				_ = unmapTable()
			}
		}()
		mask = uint64(size - 1)
		for _, path := range paths {
			reader, err := seqio.OpenPath(path)
			if err != nil {
				return err
			}
			for {
				rec, err := reader.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					closeIfPossible(reader)
					return err
				}
				klen := 0
				if len(rec.Seq) >= k {
					klen = len(rec.Seq) - k + 1
				}
				if klen <= 1 {
					continue
				}
				for i := 1; i < klen; i++ {
					key, ok, err := canonicalKmer(rec.Seq[i : i+k])
					if err != nil {
						closeIfPossible(reader)
						return err
					}
					if !ok {
						continue
					}
					slot := lookupRuntimeSlot(tableKeys, mask, key)
					tableKeys[slot] = key
				}
			}
			closeIfPossible(reader)
		}
	}
	slotsFile, err := os.CreateTemp(tmpDir, "mykrobe2-slots-*.bin")
	if err != nil {
		return err
	}
	slotsPath := slotsFile.Name()
	defer os.Remove(slotsPath)
	defer slotsFile.Close()
	slotsBuf := bufio.NewWriterSize(slotsFile, 1<<20)
	probeOffsets := make([]uint32, 0, counts.probeCount+1)
	probeOffsets = append(probeOffsets, 0)
	nameOffsets := make([]uint32, 0, counts.probeCount+1)
	nameOffsets = append(nameOffsets, 0)
	nameBytes := make([]byte, 0, counts.totalNameBytes)
	kmerLengths := make([]uint32, 0, counts.probeCount)
	var probeSlotPos uint32
	var namePos uint32
	for _, path := range paths {
		reader, err := seqio.OpenPath(path)
		if err != nil {
			return err
		}
		for {
			rec, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				closeIfPossible(reader)
				return err
			}
			klen := 0
			if len(rec.Seq) >= k {
				klen = len(rec.Seq) - k + 1
			}
			kmerLengths = append(kmerLengths, uint32(klen))
			nameBytes = append(nameBytes, rec.Name...)
			namePos += uint32(len(rec.Name))
			nameOffsets = append(nameOffsets, namePos)
			if klen <= 1 {
				probeOffsets = append(probeOffsets, probeSlotPos)
				continue
			}
			if err := binary.Write(slotsBuf, binary.LittleEndian, ^uint32(0)); err != nil {
				closeIfPossible(reader)
				return err
			}
			probeSlotPos++
			for i := 1; i < klen; i++ {
				slot := ^uint32(0)
				key, ok, err := canonicalKmer(rec.Seq[i : i+k])
				if err != nil {
					closeIfPossible(reader)
					return err
				}
				if ok {
					slot = lookupRuntimeSlot(tableKeys, mask, key)
				}
				if err := binary.Write(slotsBuf, binary.LittleEndian, slot); err != nil {
					closeIfPossible(reader)
					return err
				}
				probeSlotPos++
			}
			probeOffsets = append(probeOffsets, probeSlotPos)
		}
		closeIfPossible(reader)
	}
	if err := slotsBuf.Flush(); err != nil {
		return err
	}
	if err := slotsFile.Sync(); err != nil {
		return err
	}
	if _, err := slotsFile.Seek(0, io.SeekStart); err != nil {
		return err
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	outBuf := bufio.NewWriterSize(out, 1<<20)
	if _, err := outBuf.Write([]byte(runtimeIndexMagic)); err != nil {
		return err
	}
	header := [6]uint32{
		uint32(k),
		uint32(len(tableKeys)),
		uint32(counts.probeCount + 1),
		uint32(counts.totalSlots),
		uint32(counts.probeCount + 1),
		uint32(counts.totalNameBytes),
	}
	if err := binary.Write(outBuf, binary.LittleEndian, header[:]); err != nil {
		return err
	}
	if _, err := outBuf.Write(make([]byte, alignPad(len(runtimeIndexMagic)+len(header)*4, 8))); err != nil {
		return err
	}
	if size > 0 {
		if _, err := tableFile.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if err := unmapTable(); err != nil {
			return err
		}
		tableMapped = false
		if _, err := tableFile.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if _, err := io.Copy(outBuf, tableFile); err != nil {
			return err
		}
	}
	if _, err := outBuf.Write(make([]byte, alignPad(size*8, 4))); err != nil {
		return err
	}
	if err := binary.Write(outBuf, binary.LittleEndian, probeOffsets); err != nil {
		return err
	}
	if _, err := io.Copy(outBuf, slotsFile); err != nil {
		return err
	}
	if err := binary.Write(outBuf, binary.LittleEndian, nameOffsets); err != nil {
		return err
	}
	if _, err := outBuf.Write(nameBytes); err != nil {
		return err
	}
	if _, err := outBuf.Write(make([]byte, alignPad(counts.totalNameBytes, 4))); err != nil {
		return err
	}
	if err := binary.Write(outBuf, binary.LittleEndian, kmerLengths); err != nil {
		return err
	}
	return outBuf.Flush()
}

func runtimeTableSize(count int) int {
	if count == 0 {
		return 0
	}
	size := 1
	needed := count * 2
	for size < needed {
		size <<= 1
	}
	return size
}

func createRuntimeTableFile(dir string, count int) (*os.File, string, error) {
	f, err := os.CreateTemp(dir, "mykrobe2-table-*.bin")
	if err != nil {
		return nil, "", err
	}
	path := f.Name()
	buf := make([]byte, 1<<20)
	for i := 0; i < len(buf); i += 8 {
		binary.LittleEndian.PutUint64(buf[i:], invalidPanelKmer)
	}
	remaining := count * 8
	for remaining > 0 {
		n := len(buf)
		if remaining < n {
			n = remaining
		}
		if _, err := f.Write(buf[:n]); err != nil {
			f.Close()
			return nil, "", err
		}
		remaining -= n
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, "", err
	}
	return f, path, nil
}

func SaveRuntimeIndex(path string, idx *RuntimeIndex) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write([]byte(runtimeIndexMagic)); err != nil {
		return err
	}
	header := [6]uint32{
		uint32(idx.K),
		uint32(len(idx.TableKeys)),
		uint32(len(idx.ProbeOffsets)),
		uint32(len(idx.ProbeSlots)),
		uint32(len(idx.NameOffsets)),
		uint32(len(idx.NameBytes)),
	}
	if err := binary.Write(f, binary.LittleEndian, header[:]); err != nil {
		return err
	}
	if _, err := f.Write(make([]byte, alignPad(len(runtimeIndexMagic)+len(header)*4, 8))); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, idx.TableKeys); err != nil {
		return err
	}
	if _, err := f.Write(make([]byte, alignPad(len(idx.TableKeys)*8, 4))); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, idx.ProbeOffsets); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, idx.ProbeSlots); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, idx.NameOffsets); err != nil {
		return err
	}
	if _, err := f.Write(idx.NameBytes); err != nil {
		return err
	}
	if _, err := f.Write(make([]byte, alignPad(len(idx.NameBytes), 4))); err != nil {
		return err
	}
	return binary.Write(f, binary.LittleEndian, idx.KmerLengths)
}

func parseRuntimeIndex(data []byte) (*RuntimeIndex, error) {
	minLen := len(runtimeIndexMagic) + 6*4
	if len(data) < minLen {
		return nil, fmt.Errorf("runtime index too short")
	}
	if string(data[:len(runtimeIndexMagic)]) != runtimeIndexMagic {
		return nil, fmt.Errorf("unsupported runtime index format")
	}
	offset := len(runtimeIndexMagic)
	var header [6]uint32
	for i := range header {
		header[i] = binary.LittleEndian.Uint32(data[offset:])
		offset += 4
	}
	offset += alignPad(offset, 8)
	k := int(header[0])
	nTable := int(header[1])
	nProbeOffsets := int(header[2])
	nProbeSlots := int(header[3])
	nNameOffsets := int(header[4])
	nNameBytes := int(header[5])
	need := offset + nTable*8
	if need > len(data) {
		return nil, fmt.Errorf("runtime index truncated table")
	}
	table := unsafe.Slice((*uint64)(unsafe.Pointer(&data[offset])), nTable)
	offset = need
	offset += alignPad(nTable*8, 4)
	need = offset + nProbeOffsets*4
	if need > len(data) {
		return nil, fmt.Errorf("runtime index truncated probe offsets")
	}
	probeOffsets := unsafe.Slice((*uint32)(unsafe.Pointer(&data[offset])), nProbeOffsets)
	offset = need
	need = offset + nProbeSlots*4
	if need > len(data) {
		return nil, fmt.Errorf("runtime index truncated probe slots")
	}
	probeSlots := unsafe.Slice((*uint32)(unsafe.Pointer(&data[offset])), nProbeSlots)
	offset = need
	need = offset + nNameOffsets*4
	if need > len(data) {
		return nil, fmt.Errorf("runtime index truncated name offsets")
	}
	nameOffsets := unsafe.Slice((*uint32)(unsafe.Pointer(&data[offset])), nNameOffsets)
	offset = need
	need = offset + nNameBytes
	if need > len(data) {
		return nil, fmt.Errorf("runtime index truncated name bytes")
	}
	nameBytes := data[offset:need]
	offset = need
	offset += alignPad(nNameBytes, 4)
	nKmerLengths := nProbeOffsets - 1
	need = offset + nKmerLengths*4
	if need > len(data) {
		return nil, fmt.Errorf("runtime index truncated kmer lengths")
	}
	kmerLengths := unsafe.Slice((*uint32)(unsafe.Pointer(&data[offset])), nKmerLengths)
	return &RuntimeIndex{
		K:            k,
		TableKeys:    table,
		ProbeOffsets: probeOffsets,
		ProbeSlots:   probeSlots,
		NameOffsets:  nameOffsets,
		NameBytes:    nameBytes,
		KmerLengths:  kmerLengths,
		mappedData:   data,
	}, nil
}

func LoadRuntimeIndexBytes(data []byte) (*RuntimeIndex, error) {
	idx, err := parseRuntimeIndex(data)
	if err != nil {
		return nil, err
	}
	idx.mappedData = nil
	return idx, nil
}

func alignPad(size, align int) int {
	rem := size % align
	if rem == 0 {
		return 0
	}
	return align - rem
}

func buildRuntimeTable(kmers []uint64) ([]uint64, uint64) {
	if len(kmers) == 0 {
		return nil, 0
	}
	table, mask := buildRuntimeTableSized(len(kmers))
	for _, kmer := range kmers {
		slot := lookupRuntimeSlot(table, mask, kmer)
		table[slot] = kmer
	}
	return table, mask
}

func buildRuntimeTableSized(count int) ([]uint64, uint64) {
	if count == 0 {
		return nil, 0
	}
	size := 1
	needed := count * 2
	for size < needed {
		size <<= 1
	}
	table := make([]uint64, size)
	for i := range table {
		table[i] = invalidPanelKmer
	}
	return table, uint64(size - 1)
}

func lookupRuntimeSlot(table []uint64, mask uint64, kmer uint64) uint32 {
	i := uint32((kmer * 11400714819323198485) & mask)
	for {
		v := table[i]
		if v == invalidPanelKmer || v == kmer {
			return i
		}
		i = (i + 1) & uint32(mask)
	}
}

type RuntimeCounter struct {
	k      int
	keys   []uint64
	counts []uint32
	mask   uint64
	index  *RuntimeIndex
}

func NewRuntimeCounter(idx *RuntimeIndex) (*RuntimeCounter, error) {
	if idx.K < 1 || idx.K > 31 {
		return nil, fmt.Errorf("k must be in range 1..31, got %d", idx.K)
	}
	mask := uint64(0)
	if len(idx.TableKeys) > 0 {
		mask = uint64(len(idx.TableKeys) - 1)
	}
	return &RuntimeCounter{
		k:      idx.K,
		keys:   idx.TableKeys,
		counts: make([]uint32, len(idx.TableKeys)),
		mask:   mask,
		index:  idx,
	}, nil
}

func (c *RuntimeCounter) AddPath(path string) error {
	reader, err := seqio.OpenPath(path)
	if err != nil {
		return err
	}
	defer closeIfPossible(reader)
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		c.AddSequence(rec.Seq)
	}
}

func (c *RuntimeCounter) AddSequence(seq []byte) {
	if len(seq) < c.k || len(c.keys) == 0 {
		return
	}
	var (
		fwd   uint64
		rev   uint64
		valid int
		mask  = uint64(1<<(2*c.k)) - 1
	)
	for _, base := range seq {
		code, ok := encodeBase(base)
		if !ok {
			fwd, rev, valid = 0, 0, 0
			continue
		}
		fwd = ((fwd << 2) | uint64(code)) & mask
		rev = (rev >> 2) | (uint64(code^0x3) << (2 * (c.k - 1)))
		if valid < c.k {
			valid++
		}
		if valid < c.k {
			continue
		}
		key := fwd
		if rev < key {
			key = rev
		}
		slot := lookupRuntimeSlot(c.keys, c.mask, key)
		if c.keys[slot] == invalidPanelKmer {
			continue
		}
		c.counts[slot]++
	}
}

func (c *RuntimeCounter) Summaries() []CoverageSummary {
	out := make([]CoverageSummary, 0, len(c.index.KmerLengths))
	for i := range c.index.KmerLengths {
		start := int(c.index.ProbeOffsets[i])
		end := int(c.index.ProbeOffsets[i+1])
		klen := int(c.index.KmerLengths[i])
		summary := CoverageSummary{
			Name:       c.index.ProbeName(i),
			Colour:     0,
			KmerLength: klen,
		}
		if klen <= 1 {
			out = append(out, summary)
			continue
		}
		covgs := make([]uint32, klen)
		limit := end - start
		if limit > klen {
			limit = klen
		}
		for j := 1; j < limit; j++ {
			slot := c.index.ProbeSlots[start+j]
			if slot == ^uint32(0) {
				continue
			}
			covgs[j] = c.counts[slot]
		}
		minDepth := uint32(^uint32(0))
		var nonZero float64
		for j := 1; j < klen; j++ {
			depth := covgs[j]
			summary.KmerCount += depth
			if depth == 0 {
				continue
			}
			nonZero++
			if depth < minDepth {
				minDepth = depth
			}
		}
		if minDepth == ^uint32(0) {
			minDepth = 0
		}
		sorted := slices.Clone(covgs)
		slices.Sort(sorted)
		summary.MedianDepth = medianUint32(sorted)
		summary.MinDepth = minDepth
		summary.PercentCoverage = nonZero / float64(klen-1)
		out = append(out, summary)
	}
	return out
}

func (idx *RuntimeIndex) ProbeName(i int) string {
	start := idx.NameOffsets[i]
	end := idx.NameOffsets[i+1]
	return string(idx.NameBytes[start:end])
}

func (idx *RuntimeIndex) Close() error {
	if idx == nil || idx.mappedData == nil {
		return nil
	}
	err := runtimeIndexUnmap(idx.mappedData)
	idx.mappedData = nil
	return err
}
