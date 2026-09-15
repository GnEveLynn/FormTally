package goals

import "time"

const (
	ReasonInvalidTimezone             = "invalid_timezone"
	ReasonInvalidBirthDate            = "invalid_birth_date"
	ReasonInvalidBiologicalSex        = "invalid_biological_sex"
	ReasonInvalidActivityLevel        = "invalid_activity_level"
	ReasonAgeOutOfRange               = "age_out_of_range"
	ReasonHeightOutOfRange            = "height_out_of_range"
	ReasonWeightOutOfRange            = "weight_out_of_range"
	ReasonHealthContextRequiresManual = "health_context_requires_manual"
)

type EligibilityResult struct {
	Eligible bool
	AgeYears int
	Reasons  []string
}

func EvaluateEligibility(input CalculationInput) EligibilityResult {
	result := EligibilityResult{}
	location, err := time.LoadLocation(input.Timezone)
	if err != nil {
		result.Reasons = append(result.Reasons, ReasonInvalidTimezone)
	}
	birthDate, birthErr := time.Parse("2006-01-02", input.BirthDate)
	if birthErr != nil || input.At.IsZero() {
		result.Reasons = append(result.Reasons, ReasonInvalidBirthDate)
	}
	if input.BiologicalSex != SexMale && input.BiologicalSex != SexFemale {
		result.Reasons = append(result.Reasons, ReasonInvalidBiologicalSex)
	}
	if _, ok := activityMultipliersV2[input.ActivityLevel]; !ok {
		result.Reasons = append(result.Reasons, ReasonInvalidActivityLevel)
	}
	if err == nil && birthErr == nil && !input.At.IsZero() {
		localDate := input.At.In(location)
		result.AgeYears = localDate.Year() - birthDate.Year()
		if localDate.Month() < birthDate.Month() || localDate.Month() == birthDate.Month() && localDate.Day() < birthDate.Day() {
			result.AgeYears--
		}
		if result.AgeYears < 18 || result.AgeYears > 100 {
			result.Reasons = append(result.Reasons, ReasonAgeOutOfRange)
		}
	}
	if input.HeightCm < 100 || input.HeightCm > 250 {
		result.Reasons = append(result.Reasons, ReasonHeightOutOfRange)
	}
	if input.WeightKg < 25 || input.WeightKg > 350 {
		result.Reasons = append(result.Reasons, ReasonWeightOutOfRange)
	}
	if input.HealthContext.Pregnant || input.HealthContext.Breastfeeding || input.HealthContext.ClinicalDietRequired {
		result.Reasons = append(result.Reasons, ReasonHealthContextRequiresManual)
	}
	result.Eligible = len(result.Reasons) == 0
	return result
}
