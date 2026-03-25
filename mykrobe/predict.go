package mykrobe

import (
	"encoding/json"
	"math"
	"strings"
)

type SusceptibilityResult struct {
	Susceptibility map[string]map[string]any `json:"susceptibility"`
}

func SusceptibilityResultFromJSON(s string) (SusceptibilityResult, error) {
	var out SusceptibilityResult
	err := json.Unmarshal([]byte(s), &out)
	return out, err
}

func (r SusceptibilityResult) Diff(other SusceptibilityResult) map[string]map[string][2]string {
	diff := map[string]map[string][2]string{}
	drugs := Unique(append(r.Drugs(), other.Drugs()...))
	for _, drug := range drugs {
		p1 := "NA"
		p2 := "NA"
		if v, ok := r.Susceptibility[drug]; ok {
			p1 = v["predict"].(string)
		}
		if v, ok := other.Susceptibility[drug]; ok {
			p2 = v["predict"].(string)
		}
		if p1 != p2 {
			diff[drug] = map[string][2]string{"predict": {p1, p2}}
		}
	}
	return diff
}

func (r SusceptibilityResult) Drugs() []string {
	out := make([]string, 0, len(r.Susceptibility))
	for k := range r.Susceptibility {
		out = append(out, k)
	}
	return out
}

type TBPredictor struct {
	VariantCalls            map[string]Call
	CalledGenes             map[string]Call
	VariantToResistanceDrug map[string][]string
	ResistancePredictions   map[string]map[string]any
	DepthThreshold          float64
	IgnoreFiltered          bool
	IgnoreMinorCalls        bool
	copyNumberThresholds    map[string]float64
}

func NewTBPredictor(variantCalls map[string]Call, calledGenes map[string]Call, variantToResistancePath string) (*TBPredictor, error) {
	m := map[string][]string{}
	if err := LoadJSON(variantToResistancePath, &m); err != nil {
		return nil, err
	}
	p := &TBPredictor{
		VariantCalls:            variantCalls,
		CalledGenes:             calledGenes,
		VariantToResistanceDrug: m,
		DepthThreshold:          3,
		IgnoreFiltered:          true,
		copyNumberThresholds: map[string]float64{
			"ermA": 0.19, "ermB": 0.19, "ermC": 0.19, "ermT": 0.19, "ermY": 0.19,
			"fusA": 0.03, "fusC": 0.03, "aacAaphD": 0.04, "mecA": 0.06, "mupA": 0.21, "blaZ": 0.04, "tetK": 0.13,
		},
	}
	p.initPredictions()
	return p, nil
}

func (p *TBPredictor) initPredictions() {
	drugs := []string{}
	for _, xs := range p.VariantToResistanceDrug {
		drugs = append(drugs, xs...)
	}
	p.ResistancePredictions = map[string]map[string]any{}
	for _, drug := range Unique(drugs) {
		p.ResistancePredictions[drug] = map[string]any{"predict": "N"}
	}
}

func CopyNumber(call Call) float64 {
	coverage := call.Info["coverage"].(map[string]any)
	if alt, ok := coverage["alternate"]; ok {
		alternateDepth := alt.(map[string]any)["median_depth"].(float64)
		wtDepth := coverage["reference"].(map[string]any)["median_depth"].(float64)
		return math.Round((alternateDepth/(alternateDepth+wtDepth))*100) / 100
	}
	alternateDepth := coverage["median_depth"].(float64)
	wtDepth := call.Info["expected_depths"].([]float64)[0]
	return math.Round((alternateDepth/(alternateDepth+wtDepth))*100) / 100
}

func DepthOnAlternate(call Call) float64 {
	coverage := call.Info["coverage"].(map[string]any)
	if alt, ok := coverage["alternate"]; ok {
		return alt.(map[string]any)["median_depth"].(float64)
	}
	return coverage["median_depth"].(float64)
}

func IsFiltered(call Call) bool {
	filters, _ := call.Info["filter"].([]string)
	return len(filters) > 0
}

func (p *TBPredictor) CoverageGreaterThanThreshold(call Call, names []string) bool {
	threshold := 0.1
	for _, name := range names {
		if v, ok := p.copyNumberThresholds[name]; ok {
			threshold = v
		}
	}
	return CopyNumber(call) > threshold
}

func (p *TBPredictor) DrugsForName(name string) []string {
	if drugs, ok := p.VariantToResistanceDrug[name]; ok {
		return drugs
	}
	if i := strings.IndexByte(name, '-'); i >= 0 {
		if drugs, ok := p.VariantToResistanceDrug[name[:i]]; ok {
			return drugs
		}
	}
	if name != strings.ToLower(name) {
		return p.DrugsForName(strings.ToLower(name))
	}
	return nil
}

func (p *TBPredictor) ResistancePrediction(call Call, names []string) string {
	sum := 0
	for _, gt := range call.Genotype {
		sum += gt
	}
	switch sum {
	case 2:
		if (IsFiltered(call) && p.IgnoreFiltered) || DepthOnAlternate(call) < p.DepthThreshold {
			return "N"
		}
		if p.CoverageGreaterThanThreshold(call, names) {
			return "R"
		}
		return "S"
	case 1:
		if (IsFiltered(call) && p.IgnoreFiltered) || DepthOnAlternate(call) < p.DepthThreshold {
			return "N"
		}
		if p.CoverageGreaterThanThreshold(call, names) && !p.IgnoreMinorCalls {
			return "r"
		}
		return "S"
	case 0:
		return "S"
	default:
		return "N"
	}
}

func (p *TBPredictor) Run() SusceptibilityResult {
	for alleleName, call := range p.VariantCalls {
		p.updateResistancePrediction(alleleName, call)
	}
	for geneName, call := range p.CalledGenes {
		p.updateResistancePrediction(geneName, call)
	}
	return SusceptibilityResult{Susceptibility: p.ResistancePredictions}
}

func (p *TBPredictor) updateResistancePrediction(alleleName string, call Call) {
	for _, name := range p.namesForAllele(alleleName) {
		drugs := p.DrugsForName(name)
		pred := p.ResistancePrediction(call, p.namesForAllele(alleleName))
		for _, drug := range drugs {
			current := p.ResistancePredictions[drug]["predict"].(string)
			switch current {
			case "N":
				p.ResistancePredictions[drug]["predict"] = pred
			case "I", "S":
				if pred == "r" || pred == "R" {
					p.ResistancePredictions[drug]["predict"] = pred
				}
			case "r":
				if pred == "R" {
					p.ResistancePredictions[drug]["predict"] = pred
				}
			}
			if pred == "r" || pred == "R" {
				stored := call
				stored.Variant = nil
				calledBy, ok := p.ResistancePredictions[drug]["called_by"].(map[string]Call)
				if !ok {
					calledBy = map[string]Call{}
				}
				calledBy[alleleName] = stored
				p.ResistancePredictions[drug]["called_by"] = calledBy
			}
		}
	}
}

func (p *TBPredictor) namesForAllele(alleleName string) []string {
	params := GetParams(alleleName)
	names := []string{}
	if params["mut"] != "" {
		names = append(names, params["gene"]+"_"+params["mut"])
	}
	base := strings.Split(strings.Split(alleleName, "?")[0], "-")[0]
	if base != "" {
		names = append(names, base)
	}
	return names
}
