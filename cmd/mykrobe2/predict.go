package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/martinghunt/mykrobe2/mccortex"
	"github.com/martinghunt/mykrobe2/mykrobe"
)

func runPredict(opts *predictOptions) error {
	if opts.output == "" {
		return fmt.Errorf("predict requires --seq, a panel source, and --output")
	}
	progressWriter, err := openGUIProgressWriter(opts.guiProgressFile)
	if err != nil {
		return err
	}
	if progressWriter != nil {
		defer progressWriter.Close()
	}
	var progress mykrobe.PredictProgressFunc
	if progressWriter != nil {
		progress = progressWriter.Report
	}
	result, err := mykrobe.RunTBPredict(mykrobe.PredictRunOptions{
		Sample:               opts.sample,
		SeqPaths:             opts.seqPaths,
		IndexPath:            opts.indexPath,
		PanelArg:             opts.panelArg,
		MapPath:              opts.mapPath,
		LineagePath:          opts.lineagePath,
		PanelsDir:            opts.panelsDir,
		Species:              opts.species,
		Model:                opts.model,
		Ploidy:               opts.ploidy,
		K:                    opts.k,
		ExpectedDepth:        opts.expectedDepth,
		MinDepth:             opts.minDepth,
		ErrorRate:            opts.errorRate,
		MinorFreq:            opts.minorFreq,
		MinPropExpectedDepth: opts.minPropExpectedDepth,
		MinVariantConf:       opts.minVariantConf,
		MinGeneConf:          opts.minGeneConf,
		ReportAllCalls:       opts.reportAllCalls,
		IgnoreMinorCalls:     opts.ignoreMinorCalls,
		NCBINames:            opts.ncbiNames,
		ONT:                  opts.ont,
		GuessSequenceMethod:  opts.guessSequenceMethod,
		ConfPercentCutoff:    opts.confPercentCutoff,
		Progress:             progress,
	})
	if err != nil {
		return err
	}
	if progress != nil {
		progress(mykrobe.PredictProgressEvent{Stage: mykrobe.PredictStagePreparingResults, Message: "Preparing results"})
	}
	if opts.writeCovgs != "" {
		f, err := os.Create(opts.writeCovgs)
		if err != nil {
			return err
		}
		if err := mccortex.WriteCoverageTSVWithHeader(f, result.CoverageSummaries); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}

	err = writePredictOutput(opts, result.Output)
	if err == nil && progress != nil {
		progress(mykrobe.PredictProgressEvent{Stage: mykrobe.PredictStageComplete, Message: "Analysis complete", Fraction: 1, Determinate: true})
	}
	return err
}

func writePredictOutput(opts *predictOptions, output map[string]any) error {
	f, err := os.Create(opts.output)
	if err != nil {
		return err
	}
	defer f.Close()
	switch opts.outputFormat {
	case "json":
		return mykrobe.WriteJSONLikePython(f, output, "  ")
	case "csv":
		_, err := f.WriteString(formatCSV(output))
		return err
	case "json_and_csv":
		if err := mykrobe.WriteJSONLikePython(f, output, "  "); err != nil {
			return err
		}
		return os.WriteFile(opts.output+".csv", []byte(formatCSV(output)), 0o644)
	default:
		return fmt.Errorf("output format must be one of csv,json,json_and_csv")
	}
}

type guiProgressWriter struct {
	file    *os.File
	encoder *json.Encoder
}

func openGUIProgressWriter(path string) (*guiProgressWriter, error) {
	if path == "" {
		return nil, nil
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open GUI progress file: %w", err)
	}
	return &guiProgressWriter{file: file, encoder: json.NewEncoder(file)}, nil
}

func (w *guiProgressWriter) Report(event mykrobe.PredictProgressEvent) {
	_ = w.encoder.Encode(event)
}

func (w *guiProgressWriter) Close() error {
	return w.file.Close()
}

func formatCSV(out map[string]any) string {
	var b bytes.Buffer
	b.WriteString("sample,drug,predict\n")
	samples := make([]string, 0, len(out))
	for sample := range out {
		samples = append(samples, sample)
	}
	sort.Strings(samples)
	for _, sample := range samples {
		sampleDict, _ := out[sample].(map[string]any)
		susc, _ := sampleDict["susceptibility"].(map[string]map[string]any)
		drugs := make([]string, 0, len(susc))
		for drug := range susc {
			drugs = append(drugs, drug)
		}
		sort.Strings(drugs)
		for _, drug := range drugs {
			b.WriteString(strings.Join([]string{sample, drug, fmt.Sprint(susc[drug]["predict"])}, ","))
			b.WriteByte('\n')
		}
	}
	return b.String()
}
