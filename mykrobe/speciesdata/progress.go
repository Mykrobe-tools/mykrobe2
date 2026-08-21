package speciesdata

import "io"

type PanelProgressStage string

const (
	PanelStageChecking    PanelProgressStage = "checking"
	PanelStageDownloading PanelProgressStage = "downloading"
	PanelStageExtracting  PanelProgressStage = "extracting"
	PanelStageIndexing    PanelProgressStage = "indexing"
	PanelStageFinishing   PanelProgressStage = "finishing"
	PanelStageComplete    PanelProgressStage = "complete"
)

// PanelProgressEvent describes the current panel installation phase and, when
// determinate, progress through that phase as a fraction in the range 0..1.
type PanelProgressEvent struct {
	Stage       PanelProgressStage `json:"stage"`
	Message     string             `json:"message"`
	Fraction    float64            `json:"fraction"`
	Determinate bool               `json:"determinate"`
}

type PanelProgressFunc func(PanelProgressEvent)

func reportPanelProgress(report PanelProgressFunc, stage PanelProgressStage, message string, fraction float64, determinate bool) {
	if report == nil {
		return
	}
	report(PanelProgressEvent{
		Stage:       stage,
		Message:     message,
		Fraction:    fraction,
		Determinate: determinate,
	})
}

type panelDownloadProgressReader struct {
	reader       io.Reader
	total        int64
	read         int64
	lastFraction float64
	report       PanelProgressFunc
}

func (r *panelDownloadProgressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += int64(n)
	if r.total > 0 {
		fraction := min(1, float64(r.read)/float64(r.total))
		if fraction-r.lastFraction >= 0.005 || fraction >= 1 {
			r.lastFraction = fraction
			reportPanelProgress(r.report, PanelStageDownloading, "Downloading panel data", fraction, true)
		}
	}
	return n, err
}
