package aieval

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"time"
)

const (
	StatusPass    = "pass"
	StatusFail    = "fail"
	StatusBlocked = "blocked"
)

type Nutrition struct {
	EnergyKcal   float64 `json:"energyKcal"`
	ProteinGrams float64 `json:"proteinGrams"`
	CarbGrams    float64 `json:"carbGrams"`
	FatGrams     float64 `json:"fatGrams"`
}

type Evaluation struct {
	Structured          bool    `json:"structured"`
	EstimatedEnergyKcal float64 `json:"estimatedEnergyKcal,omitempty"`
	DurationMS          int64   `json:"durationMs"`
	CostUSD             float64 `json:"costUsd"`
}

type Sample struct {
	ID           string      `json:"id"`
	ImagePath    string      `json:"imagePath"`
	Category     string      `json:"category"`
	HiddenOil    bool        `json:"hiddenOil"`
	WeighedGrams float64     `json:"weighedGrams"`
	Reference    Nutrition   `json:"reference"`
	Evaluation   *Evaluation `json:"evaluation,omitempty"`
}

type Manifest struct {
	Version       string    `json:"version"`
	Model         string    `json:"model"`
	PromptVersion string    `json:"promptVersion"`
	EvaluatedAt   time.Time `json:"evaluatedAt"`
	Samples       []Sample  `json:"samples"`
}

type Report struct {
	Model                 string    `json:"model"`
	PromptVersion         string    `json:"promptVersion"`
	EvaluatedAt           time.Time `json:"evaluatedAt"`
	SampleCount           int       `json:"sampleCount"`
	EvaluatedCount        int       `json:"evaluatedCount"`
	StructuredCount       int       `json:"structuredCount"`
	StructuredSuccessRate float64   `json:"structuredSuccessRate"`
	MedianAPE             float64   `json:"medianApe"`
	P90APE                float64   `json:"p90Ape"`
	TotalDurationMS       int64     `json:"totalDurationMs"`
	MeanDurationMS        float64   `json:"meanDurationMs"`
	TotalCostUSD          float64   `json:"totalCostUsd"`
}

type Gate struct {
	Status   string   `json:"status"`
	Blockers []string `json:"blockers"`
	Failures []string `json:"failures"`
}

func Evaluate(manifest Manifest) (Report, error) {
	if manifest.Version == "" || manifest.Model == "" || manifest.PromptVersion == "" || manifest.EvaluatedAt.IsZero() {
		return Report{}, errors.New("manifest provenance is incomplete")
	}
	if len(manifest.Samples) == 0 {
		return Report{}, errors.New("manifest samples are empty")
	}
	report := Report{Model: manifest.Model, PromptVersion: manifest.PromptVersion, EvaluatedAt: manifest.EvaluatedAt, SampleCount: len(manifest.Samples)}
	ids := make(map[string]struct{}, len(manifest.Samples))
	apes := make([]float64, 0, len(manifest.Samples))
	for index, sample := range manifest.Samples {
		if sample.ID == "" || sample.ImagePath == "" || sample.Category == "" || sample.WeighedGrams <= 0 || invalidPositive(sample.Reference.EnergyKcal) {
			return Report{}, fmt.Errorf("sample %d is invalid", index)
		}
		if _, exists := ids[sample.ID]; exists {
			return Report{}, fmt.Errorf("duplicate sample id %q", sample.ID)
		}
		ids[sample.ID] = struct{}{}
		if sample.Evaluation == nil {
			continue
		}
		evaluation := sample.Evaluation
		if evaluation.DurationMS < 0 || evaluation.CostUSD < 0 || math.IsNaN(evaluation.CostUSD) || math.IsInf(evaluation.CostUSD, 0) {
			return Report{}, fmt.Errorf("sample %q has invalid evaluation metrics", sample.ID)
		}
		if evaluation.Structured && invalidPositive(evaluation.EstimatedEnergyKcal) {
			return Report{}, fmt.Errorf("sample %q has invalid estimated energy", sample.ID)
		}
		report.EvaluatedCount++
		report.TotalDurationMS += evaluation.DurationMS
		report.TotalCostUSD += evaluation.CostUSD
		if evaluation.Structured {
			report.StructuredCount++
			apes = append(apes, math.Abs(evaluation.EstimatedEnergyKcal-sample.Reference.EnergyKcal)/sample.Reference.EnergyKcal)
		}
	}
	if report.EvaluatedCount > 0 {
		report.StructuredSuccessRate = float64(report.StructuredCount) / float64(report.EvaluatedCount)
		report.MeanDurationMS = float64(report.TotalDurationMS) / float64(report.EvaluatedCount)
	}
	if len(apes) > 0 {
		slices.Sort(apes)
		report.MedianAPE = percentile(apes, .5)
		report.P90APE = percentile(apes, .9)
	}
	return report, nil
}

func Assess(report Report) Gate {
	gate := Gate{Status: StatusPass, Blockers: []string{}, Failures: []string{}}
	if report.SampleCount < 50 {
		gate.Blockers = append(gate.Blockers, "真实模型评测样本必须不少于 50 餐")
	}
	if report.EvaluatedCount < report.SampleCount {
		gate.Blockers = append(gate.Blockers, "manifest 中仍有样本缺少真实模型评测结果")
	}
	if len(gate.Blockers) > 0 {
		gate.Status = StatusBlocked
		return gate
	}
	if report.StructuredSuccessRate < .95 {
		gate.Failures = append(gate.Failures, "结构化成功率低于 95%")
	}
	if report.MedianAPE > .30 {
		gate.Failures = append(gate.Failures, "整餐热量 MdAPE 高于 30%")
	}
	if report.P90APE > .60 {
		gate.Failures = append(gate.Failures, "整餐热量 p90 APE 高于 60%")
	}
	if len(gate.Failures) > 0 {
		gate.Status = StatusFail
	}
	return gate
}

func invalidPositive(value float64) bool {
	return value <= 0 || math.IsNaN(value) || math.IsInf(value, 0)
}

func percentile(sorted []float64, percentile float64) float64 {
	position := float64(len(sorted)-1) * percentile
	lower, upper := int(math.Floor(position)), int(math.Ceil(position))
	if lower == upper {
		return sorted[lower]
	}
	return sorted[lower] + (sorted[upper]-sorted[lower])*(position-float64(lower))
}
