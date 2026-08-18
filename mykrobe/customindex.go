package mykrobe

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mykrobe-tools/mykrobe2/mccortex"
)

const customIndexMagic = "MYKROBE2IDX1"

type customIndexBlob struct {
	Offset uint64 `json:"offset"`
	Length uint64 `json:"length"`
}

type customIndexManifest struct {
	ProbeSets    []string         `json:"probe_sets,omitempty"`
	PanelVersion string           `json:"panel_version,omitempty"`
	AMR          *customIndexBlob `json:"amr,omitempty"`
	Lineage      *customIndexBlob `json:"lineage,omitempty"`
}

type CustomIndexBundle struct {
	RuntimeIndex        *mccortex.RuntimeIndex
	ProbeSets           []string
	PanelVersion        string
	VariantToResistance []byte
	Lineage             []byte

	rawData []byte
}

func BuildCustomIndex(outputPath string, k int, fastaPaths []string, amrPath, lineagePath string) error {
	tmpDir := filepath.Dir(outputPath)
	tmpRuntime, err := os.CreateTemp(tmpDir, "mykrobe2-custom-runtime-*.panelindex")
	if err != nil {
		return err
	}
	tmpRuntimePath := tmpRuntime.Name()
	if err := tmpRuntime.Close(); err != nil {
		return err
	}
	defer os.Remove(tmpRuntimePath)

	if err := mccortex.BuildRuntimeIndexFile(tmpRuntimePath, k, fastaPaths); err != nil {
		return err
	}
	runtimeBytes, err := os.ReadFile(tmpRuntimePath)
	if err != nil {
		return err
	}

	manifest := customIndexManifest{
		ProbeSets:    append([]string(nil), fastaPaths...),
		PanelVersion: "custom",
	}
	payload := make([]byte, 0)
	appendBlob := func(path string) (*customIndexBlob, error) {
		if path == "" {
			return nil, nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		offset := uint64(len(runtimeBytes) + len(payload))
		payload = append(payload, data...)
		return &customIndexBlob{Offset: offset, Length: uint64(len(data))}, nil
	}
	if manifest.AMR, err = appendBlob(amrPath); err != nil {
		return err
	}
	if manifest.Lineage, err = appendBlob(lineagePath); err != nil {
		return err
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(runtimeBytes); err != nil {
		return err
	}
	if _, err := f.Write(payload); err != nil {
		return err
	}
	if _, err := f.Write(manifestBytes); err != nil {
		return err
	}
	var sizeBuf [8]byte
	binary.LittleEndian.PutUint64(sizeBuf[:], uint64(len(manifestBytes)))
	if _, err := f.Write(sizeBuf[:]); err != nil {
		return err
	}
	_, err = f.WriteString(customIndexMagic)
	return err
}

func LoadCustomIndex(path string) (*CustomIndexBundle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	idxData := data
	manifest := customIndexManifest{
		ProbeSets:    []string{path},
		PanelVersion: "custom",
	}
	if len(data) >= len(customIndexMagic)+8 && string(data[len(data)-len(customIndexMagic):]) == customIndexMagic {
		manifestLenPos := len(data) - len(customIndexMagic) - 8
		manifestLen := binary.LittleEndian.Uint64(data[manifestLenPos:])
		if manifestLen > uint64(manifestLenPos) {
			return nil, fmt.Errorf("custom index manifest truncated")
		}
		manifestPos := manifestLenPos - int(manifestLen)
		if err := json.Unmarshal(data[manifestPos:manifestLenPos], &manifest); err != nil {
			return nil, err
		}
		switch {
		case manifest.AMR != nil && int(manifest.AMR.Offset+manifest.AMR.Length) > len(data):
			return nil, fmt.Errorf("custom index amr blob truncated")
		case manifest.Lineage != nil && int(manifest.Lineage.Offset+manifest.Lineage.Length) > len(data):
			return nil, fmt.Errorf("custom index lineage blob truncated")
		}
		idxData = data[:manifestPos]
	}
	idx, err := mccortex.LoadRuntimeIndexBytes(idxData)
	if err != nil {
		return nil, err
	}
	out := &CustomIndexBundle{
		RuntimeIndex: idx,
		ProbeSets:    append([]string(nil), manifest.ProbeSets...),
		PanelVersion: manifest.PanelVersion,
		rawData:      data,
	}
	if manifest.AMR != nil {
		start := manifest.AMR.Offset
		end := start + manifest.AMR.Length
		out.VariantToResistance = data[start:end]
	}
	if manifest.Lineage != nil {
		start := manifest.Lineage.Offset
		end := start + manifest.Lineage.Length
		out.Lineage = data[start:end]
	}
	if out.PanelVersion == "" {
		out.PanelVersion = "custom"
	}
	if len(out.ProbeSets) == 0 {
		out.ProbeSets = []string{path}
	}
	return out, nil
}

func (b *CustomIndexBundle) Close() error {
	if b == nil || b.RuntimeIndex == nil {
		return nil
	}
	err := b.RuntimeIndex.Close()
	b.rawData = nil
	return err
}
