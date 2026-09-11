package goals

import (
	"testing"
	"time"
)

func eligibilityInput(birthDate string, heightCm, weightKg float64) CalculationInput {
	input := fixedInput()
	input.BirthDate = birthDate
	input.HeightCm = heightCm
	input.WeightKg = weightKg
	input.At = time.Date(2026, 9, 11, 4, 0, 0, 0, time.UTC)
	return input
}

func TestEligibilityAcceptsInclusiveBoundaries(t *testing.T) {
	for _, input := range []CalculationInput{
		eligibilityInput("2008-09-11", 100, 25),
		eligibilityInput("1926-09-11", 250, 350),
	} {
		if result := EvaluateEligibility(input); !result.Eligible {
			t.Fatalf("input %+v should be eligible, reasons = %v", input, result.Reasons)
		}
	}
}

func TestEligibilityRejectsOutOfRangeInputs(t *testing.T) {
	tests := []struct {
		name   string
		input  CalculationInput
		reason string
	}{
		{"under 18", eligibilityInput("2008-09-12", 178, 72.5), ReasonAgeOutOfRange},
		{"over 100", eligibilityInput("1925-09-11", 178, 72.5), ReasonAgeOutOfRange},
		{"height low", eligibilityInput("1995-06-18", 99.9, 72.5), ReasonHeightOutOfRange},
		{"height high", eligibilityInput("1995-06-18", 250.1, 72.5), ReasonHeightOutOfRange},
		{"weight low", eligibilityInput("1995-06-18", 178, 24.9), ReasonWeightOutOfRange},
		{"weight high", eligibilityInput("1995-06-18", 178, 350.1), ReasonWeightOutOfRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluateEligibility(tt.input)
			if result.Eligible || !contains(result.Reasons, tt.reason) {
				t.Fatalf("eligibility = %+v, want reason %q", result, tt.reason)
			}
		})
	}
}

func TestAgeUsesTargetLocalDate(t *testing.T) {
	instant := time.Date(2026, 9, 11, 2, 0, 0, 0, time.UTC)
	input := eligibilityInput("2008-09-11", 178, 72.5)
	input.At = instant
	input.Timezone = "Asia/Shanghai"
	if result := EvaluateEligibility(input); !result.Eligible {
		t.Fatalf("Shanghai local date should be 18th birthday: %+v", result)
	}
	input.Timezone = "America/Los_Angeles"
	if result := EvaluateEligibility(input); result.Eligible || !contains(result.Reasons, ReasonAgeOutOfRange) {
		t.Fatalf("Los Angeles local date should still be age 17: %+v", result)
	}
}

func TestEligibilityRejectsInvalidIANATimezone(t *testing.T) {
	input := eligibilityInput("1995-06-18", 178, 72.5)
	input.Timezone = "Mars/Olympus"
	result := EvaluateEligibility(input)
	if result.Eligible || !contains(result.Reasons, ReasonInvalidTimezone) {
		t.Fatalf("eligibility = %+v", result)
	}
}

func TestEligibilityRejectsSpecialHealthContexts(t *testing.T) {
	tests := []HealthContext{
		{Pregnant: true},
		{Breastfeeding: true},
		{ClinicalDietRequired: true},
	}
	for _, health := range tests {
		input := eligibilityInput("1995-06-18", 178, 72.5)
		input.HealthContext = health
		result := EvaluateEligibility(input)
		if result.Eligible || !contains(result.Reasons, ReasonHealthContextRequiresManual) {
			t.Fatalf("health context %+v should require manual goal: %+v", health, result)
		}
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
