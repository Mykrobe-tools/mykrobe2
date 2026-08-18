package mykrobe

// PredictProgressStage identifies a stable phase of the prediction workflow.
type PredictProgressStage string

const (
	PredictStageLoadingPanel         PredictProgressStage = "loading_panel"
	PredictStageProcessingReads      PredictProgressStage = "processing_reads"
	PredictStageCalculatingCoverage  PredictProgressStage = "calculating_coverage"
	PredictStageIdentifyingSpecies   PredictProgressStage = "identifying_species"
	PredictStagePredictingResistance PredictProgressStage = "predicting_resistance"
	PredictStagePreparingResults     PredictProgressStage = "preparing_results"
	PredictStageComplete             PredictProgressStage = "complete"
)

// PredictProgressEvent describes the current prediction phase and, when
// determinate, progress through that phase as a fraction in the range 0..1.
type PredictProgressEvent struct {
	Stage       PredictProgressStage `json:"stage"`
	Message     string               `json:"message"`
	Fraction    float64              `json:"fraction"`
	Determinate bool                 `json:"determinate"`
}

// PredictProgressFunc receives optional progress updates from RunTBPredict.
type PredictProgressFunc func(PredictProgressEvent)

func reportPredictProgress(report PredictProgressFunc, stage PredictProgressStage, message string, fraction float64, determinate bool) {
	if report == nil {
		return
	}
	report(PredictProgressEvent{
		Stage:       stage,
		Message:     message,
		Fraction:    fraction,
		Determinate: determinate,
	})
}
