package aieval

import (
	"math"
	"testing"
	"time"
)

func TestEvaluateSummarizesStructuredQualityLatencyAndCost(t *testing.T) {
	evaluatedAt := time.Date(2026, 9, 13, 1, 0, 0, 0, time.UTC)
	manifest := Manifest{Version: "v1", Model: "test-model", PromptVersion: "meal_analysis_v1", EvaluatedAt: evaluatedAt, Samples: []Sample{
		{ID: "a", ImagePath: "images/a.jpg", Category: "rice", WeighedGrams: 100, Reference: Nutrition{EnergyKcal: 100}, Evaluation: &Evaluation{Structured: true, EstimatedEnergyKcal: 100, DurationMS: 1000, CostUSD: .01}},
		{ID: "b", ImagePath: "images/b.jpg", Category: "noodles", HiddenOil: true, WeighedGrams: 200, Reference: Nutrition{EnergyKcal: 100}, Evaluation: &Evaluation{Structured: true, EstimatedEnergyKcal: 120, DurationMS: 2000, CostUSD: .02}},
		{ID: "c", ImagePath: "images/c.jpg", Category: "mixed", WeighedGrams: 300, Reference: Nutrition{EnergyKcal: 100}, Evaluation: &Evaluation{Structured: true, EstimatedEnergyKcal: 140, DurationMS: 3000, CostUSD: .03}},
		{ID: "d", ImagePath: "images/d.jpg", Category: "soup", WeighedGrams: 400, Reference: Nutrition{EnergyKcal: 100}, Evaluation: &Evaluation{Structured: false, DurationMS: 4000, CostUSD: .04}},
	}}

	report, err := Evaluate(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if report.SampleCount != 4 || report.EvaluatedCount != 4 || report.StructuredCount != 3 || report.StructuredSuccessRate != .75 {
		t.Fatalf("counts/rate = %+v", report)
	}
	if report.MedianAPE != .20 || math.Abs(report.P90APE-.36) > 1e-9 {
		t.Fatalf("errors = median %v p90 %v", report.MedianAPE, report.P90APE)
	}
	if report.TotalDurationMS != 10000 || report.MeanDurationMS != 2500 || math.Abs(report.TotalCostUSD-.10) > 1e-9 {
		t.Fatalf("duration/cost = %+v", report)
	}
	if report.Model != manifest.Model || report.PromptVersion != manifest.PromptVersion || !report.EvaluatedAt.Equal(evaluatedAt) {
		t.Fatalf("provenance = %+v", report)
	}
}

func TestEvaluateRejectsInvalidOrDuplicateSamples(t *testing.T) {
	valid := Sample{ID: "sample", ImagePath: "images/sample.jpg", Category: "mixed", WeighedGrams: 100, Reference: Nutrition{EnergyKcal: 500}, Evaluation: &Evaluation{Structured: true, EstimatedEnergyKcal: 500}}
	for name, samples := range map[string][]Sample{
		"empty":            nil,
		"duplicate id":     {valid, valid},
		"missing image":    {{ID: "sample", Category: "mixed", WeighedGrams: 100, Reference: Nutrition{EnergyKcal: 500}}},
		"zero weight":      {{ID: "sample", ImagePath: "x.jpg", Category: "mixed", Reference: Nutrition{EnergyKcal: 500}}},
		"zero reference":   {{ID: "sample", ImagePath: "x.jpg", Category: "mixed", WeighedGrams: 100}},
		"negative cost":    {{ID: "sample", ImagePath: "x.jpg", Category: "mixed", WeighedGrams: 100, Reference: Nutrition{EnergyKcal: 500}, Evaluation: &Evaluation{CostUSD: -1}}},
		"negative latency": {{ID: "sample", ImagePath: "x.jpg", Category: "mixed", WeighedGrams: 100, Reference: Nutrition{EnergyKcal: 500}, Evaluation: &Evaluation{DurationMS: -1}}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Evaluate(Manifest{Version: "v1", Model: "test-model", PromptVersion: "v1", EvaluatedAt: time.Now(), Samples: samples})
			if err == nil {
				t.Fatal("invalid manifest accepted")
			}
		})
	}
}

func TestAssessDistinguishesBlockedFailedAndPassingEvaluation(t *testing.T) {
	passing := Report{SampleCount: 50, EvaluatedCount: 50, StructuredSuccessRate: .96, MedianAPE: .30, P90APE: .60}
	if gate := Assess(passing); gate.Status != StatusPass || len(gate.Blockers) != 0 || len(gate.Failures) != 0 {
		t.Fatalf("passing gate = %+v", gate)
	}
	if gate := Assess(Report{SampleCount: 49, EvaluatedCount: 49, StructuredSuccessRate: 1}); gate.Status != StatusBlocked || len(gate.Blockers) != 1 {
		t.Fatalf("small dataset gate = %+v", gate)
	}
	if gate := Assess(Report{SampleCount: 50, EvaluatedCount: 49, StructuredSuccessRate: 1}); gate.Status != StatusBlocked || len(gate.Blockers) != 1 {
		t.Fatalf("incomplete dataset gate = %+v", gate)
	}
	for name, report := range map[string]Report{
		"structured": {SampleCount: 50, EvaluatedCount: 50, StructuredSuccessRate: .949, MedianAPE: .1, P90APE: .2},
		"median":     {SampleCount: 50, EvaluatedCount: 50, StructuredSuccessRate: 1, MedianAPE: .301, P90APE: .2},
		"p90":        {SampleCount: 50, EvaluatedCount: 50, StructuredSuccessRate: 1, MedianAPE: .1, P90APE: .601},
	} {
		t.Run(name, func(t *testing.T) {
			gate := Assess(report)
			if gate.Status != StatusFail || len(gate.Failures) == 0 {
				t.Fatalf("failing gate = %+v", gate)
			}
		})
	}
}
