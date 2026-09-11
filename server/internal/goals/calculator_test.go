package goals

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"
)

func fixedInput() CalculationInput {
	return CalculationInput{
		BiologicalSex: SexMale,
		BirthDate:     "1995-06-18",
		HeightCm:      178,
		WeightKg:      72.5,
		ActivityLevel: ActivityModerate,
		Timezone:      "Asia/Shanghai",
		At:            time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC),
		Objective:     ObjectiveFatLoss,
		Pace:          PaceStandard,
	}
}

func TestCalculateFixedExample(t *testing.T) {
	result, err := CalculateV1(fixedInput())
	if err != nil {
		t.Fatal(err)
	}

	wantTarget := NutritionTarget{EnergyKcal: 2220, ProteinGrams: 139, CarbGrams: 278, FatGrams: 62}
	if result.Target != wantTarget {
		t.Fatalf("target = %+v, want %+v", result.Target, wantTarget)
	}
	if result.Calculation.CalculationVersion != CalculationVersionV1 || result.Calculation.Method.ID != EnergyMethodV1 {
		t.Fatalf("versions = %q/%q", result.Calculation.CalculationVersion, result.Calculation.Method.ID)
	}
	if result.Calculation.Inputs.AgeYears != 31 || result.Calculation.Inputs.ActivityMultiplier != 1.55 || result.Calculation.Inputs.GoalAdjustmentPercent != -15 {
		t.Fatalf("calculation inputs = %+v", result.Calculation.Inputs)
	}
	wantSteps := []float64{1687.5, 2615.6, 2223.3, 2220}
	if len(result.Calculation.Steps) != len(wantSteps) {
		t.Fatalf("steps = %+v", result.Calculation.Steps)
	}
	for i, want := range wantSteps {
		if math.Abs(result.Calculation.Steps[i].Value-want) > 0.001 {
			t.Fatalf("step %d = %v, want %v", i, result.Calculation.Steps[i].Value, want)
		}
	}
	if result.Calculation.Steps[1].Value != 2615.6 {
		t.Fatalf("maintenance step leaks floating-point noise: %v", result.Calculation.Steps[1].Value)
	}
	if result.Calculation.Rounding.HalfRule != "half_away_from_zero" || result.Calculation.Method.SourceURL == "" || result.Calculation.Disclaimer == "" {
		t.Fatalf("incomplete explanation = %+v", result.Calculation)
	}
}

func TestCalculateVariants(t *testing.T) {
	tests := []struct {
		name      string
		sex       BiologicalSex
		activity  ActivityLevel
		objective Objective
		pace      Pace
		energy    int
	}{
		{"female maintain", SexFemale, ActivityModerate, ObjectiveMaintain, PaceNone, 2360},
		{"sedentary", SexMale, ActivitySedentary, ObjectiveMaintain, PaceNone, 2030},
		{"light", SexMale, ActivityLight, ObjectiveMaintain, PaceNone, 2320},
		{"moderate", SexMale, ActivityModerate, ObjectiveMaintain, PaceNone, 2620},
		{"high", SexMale, ActivityHigh, ObjectiveMaintain, PaceNone, 2910},
		{"very high", SexMale, ActivityVeryHigh, ObjectiveMaintain, PaceNone, 3210},
		{"fat loss slow", SexMale, ActivityModerate, ObjectiveFatLoss, PaceSlow, 2350},
		{"fat loss standard", SexMale, ActivityModerate, ObjectiveFatLoss, PaceStandard, 2220},
		{"fat loss fast", SexMale, ActivityModerate, ObjectiveFatLoss, PaceFast, 2090},
		{"muscle gain slow", SexMale, ActivityModerate, ObjectiveMuscleGain, PaceSlow, 2750},
		{"muscle gain standard", SexMale, ActivityModerate, ObjectiveMuscleGain, PaceStandard, 2880},
		{"muscle gain fast", SexMale, ActivityModerate, ObjectiveMuscleGain, PaceFast, 3010},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := fixedInput()
			input.BiologicalSex = tt.sex
			input.ActivityLevel = tt.activity
			input.Objective = tt.objective
			input.Pace = tt.pace
			result, err := CalculateV1(input)
			if err != nil {
				t.Fatal(err)
			}
			if result.Target.EnergyKcal != tt.energy {
				t.Fatalf("energy = %d, want %d", result.Target.EnergyKcal, tt.energy)
			}
		})
	}
}

func TestCalculateRejectsInvalidObjectivePaceCombination(t *testing.T) {
	input := fixedInput()
	input.Objective = ObjectiveMaintain
	input.Pace = PaceFast
	if _, err := CalculateV1(input); !errors.Is(err, ErrInvalidGoal) {
		t.Fatalf("err = %v, want ErrInvalidGoal", err)
	}
}

func TestRoundingUsesHalfAwayFromZero(t *testing.T) {
	for _, tt := range []struct {
		value, unit, want float64
	}{{15, 10, 20}, {-15, 10, -20}, {2.5, 1, 3}, {-2.5, 1, -3}} {
		if got := roundTo(tt.value, tt.unit); got != tt.want {
			t.Fatalf("roundTo(%v, %v) = %v, want %v", tt.value, tt.unit, got, tt.want)
		}
	}
}

func TestReplaySavedV1Input(t *testing.T) {
	fixture := []byte(`{
		"biologicalSex":"male","birthDate":"1995-06-18","heightCm":178,"weightKg":72.5,
		"activityLevel":"moderate","timezone":"Asia/Shanghai","at":"2026-09-10T08:00:00Z",
		"objective":"fat_loss","pace":"standard","healthContext":{}
	}`)
	var input CalculationInput
	if err := json.Unmarshal(fixture, &input); err != nil {
		t.Fatal(err)
	}
	result, err := CalculateV1(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Target != (NutritionTarget{EnergyKcal: 2220, ProteinGrams: 139, CarbGrams: 278, FatGrams: 62}) {
		t.Fatalf("replayed target = %+v", result.Target)
	}
	if got := result.Calculation.Steps[2].Value; got != 2223.3 {
		t.Fatalf("saved V1 goal step = %v, want 2223.3", got)
	}
}
