package goals

import (
	"errors"
	"time"
)

const (
	CalculationVersionV1 = "daily_nutrition_target_v1"
	EnergyMethodV1       = "mifflin_st_jeor_v1"
	MacroMethodV1        = "macro_split_50_25_25_v1"
)

var (
	ErrAutomaticGoalNotEligible = errors.New("automatic goal is not eligible")
	ErrInvalidGoal              = errors.New("invalid goal")
	ErrRevisionConflict         = errors.New("goal revision conflict")
	ErrProfileRequired          = errors.New("profile required")
)

type BiologicalSex string

const (
	SexMale   BiologicalSex = "male"
	SexFemale BiologicalSex = "female"
)

type ActivityLevel string

const (
	ActivitySedentary ActivityLevel = "sedentary"
	ActivityLight     ActivityLevel = "light"
	ActivityModerate  ActivityLevel = "moderate"
	ActivityHigh      ActivityLevel = "high"
	ActivityVeryHigh  ActivityLevel = "very_high"
)

type Objective string

const (
	ObjectiveFatLoss    Objective = "fat_loss"
	ObjectiveMaintain   Objective = "maintain"
	ObjectiveMuscleGain Objective = "muscle_gain"
)

type Pace string

const (
	PaceNone     Pace = ""
	PaceSlow     Pace = "slow"
	PaceStandard Pace = "standard"
	PaceFast     Pace = "fast"
)

type HealthContext struct {
	Pregnant             bool `json:"pregnant"`
	Breastfeeding        bool `json:"breastfeeding"`
	ClinicalDietRequired bool `json:"clinicalDietRequired"`
}

type CalculationInput struct {
	BiologicalSex BiologicalSex `json:"biologicalSex"`
	BirthDate     string        `json:"birthDate"`
	HeightCm      float64       `json:"heightCm"`
	WeightKg      float64       `json:"weightKg"`
	ActivityLevel ActivityLevel `json:"activityLevel"`
	Timezone      string        `json:"timezone"`
	At            time.Time     `json:"at"`
	Objective     Objective     `json:"objective"`
	Pace          Pace          `json:"pace"`
	HealthContext HealthContext `json:"healthContext"`
}

type NutritionTarget struct {
	EnergyKcal   int `json:"energyKcal"`
	ProteinGrams int `json:"proteinGrams"`
	CarbGrams    int `json:"carbGrams"`
	FatGrams     int `json:"fatGrams"`
}

type MethodDescription struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	SourceURL         string `json:"sourceUrl"`
	FormulaExpression string `json:"formulaExpression"`
}

type MacroMethodDescription struct {
	ID             string `json:"id"`
	CarbPercent    int    `json:"carbPercent"`
	ProteinPercent int    `json:"proteinPercent"`
	FatPercent     int    `json:"fatPercent"`
}

type CalculationInputs struct {
	BiologicalSex         BiologicalSex `json:"biologicalSex"`
	AgeYears              int           `json:"ageYears"`
	HeightCm              float64       `json:"heightCm"`
	WeightKg              float64       `json:"weightKg"`
	ActivityLevel         ActivityLevel `json:"activityLevel"`
	ActivityMultiplier    float64       `json:"activityMultiplier"`
	Objective             Objective     `json:"objective"`
	Pace                  *Pace         `json:"pace"`
	GoalAdjustmentPercent int           `json:"goalAdjustmentPercent"`
}

type CalculationStep struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type RoundingDescription struct {
	Energy   string `json:"energy"`
	Macros   string `json:"macros"`
	HalfRule string `json:"halfRule"`
}

type GoalCalculation struct {
	CalculationVersion string                 `json:"calculationVersion"`
	Method             MethodDescription      `json:"method"`
	MacroMethod        MacroMethodDescription `json:"macroMethod"`
	Inputs             CalculationInputs      `json:"inputs"`
	Steps              []CalculationStep      `json:"steps"`
	Rounding           RoundingDescription    `json:"rounding"`
	Disclaimer         string                 `json:"disclaimer"`
}

type CalculationResult struct {
	Target      NutritionTarget `json:"target"`
	Calculation GoalCalculation `json:"calculation"`
	Warnings    []string        `json:"warnings"`
}

type Mode string

const (
	ModeAutomatic Mode = "automatic"
	ModeManual    Mode = "manual"
)

type AutomaticSettings struct {
	Objective Objective `json:"objective"`
	Pace      Pace      `json:"pace,omitempty"`
}

type ManualSettings struct {
	Target NutritionTarget `json:"target"`
}

type SettingsInput struct {
	Mode             Mode               `json:"mode"`
	Automatic        *AutomaticSettings `json:"automatic"`
	Manual           *ManualSettings    `json:"manual"`
	ExpectedRevision *int               `json:"expectedRevision,omitempty"`
}

type SettingsView struct {
	Mode      Mode               `json:"mode"`
	Automatic *AutomaticSettings `json:"automatic"`
	Manual    *ManualSettings    `json:"manual"`
	Revision  int                `json:"revision"`
	UpdatedAt time.Time          `json:"updatedAt"`
}

type EffectiveTarget struct {
	EffectiveFrom string           `json:"effectiveFrom,omitempty"`
	LocalDate     string           `json:"localDate,omitempty"`
	Target        NutritionTarget  `json:"target"`
	Calculation   *GoalCalculation `json:"calculation"`
	Warnings      []string         `json:"warnings"`
}

type SaveResult struct {
	Settings        SettingsView    `json:"settings"`
	EffectiveTarget EffectiveTarget `json:"effectiveTarget"`
}

type View struct {
	Settings      *SettingsView    `json:"settings"`
	ActiveTarget  *EffectiveTarget `json:"activeTarget"`
	PendingTarget *EffectiveTarget `json:"pendingTarget"`
}
