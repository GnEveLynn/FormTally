package goals

var activityMultipliers = map[ActivityLevel]float64{
	// These factors represent total daily activity relative to resting energy,
	// from little exercise through intense training plus physical work.
	ActivitySedentary: 1.2,
	ActivityLight:     1.375,
	ActivityModerate:  1.55,
	ActivityHigh:      1.725,
	ActivityVeryHigh:  1.9,
}

var goalAdjustments = map[Objective]map[Pace]float64{
	// Product policy applies progressively larger deficits or surpluses to
	// maintenance energy; maintain intentionally has no pace.
	ObjectiveFatLoss:    {PaceSlow: -0.10, PaceStandard: -0.15, PaceFast: -0.20},
	ObjectiveMaintain:   {PaceNone: 0},
	ObjectiveMuscleGain: {PaceSlow: 0.05, PaceStandard: 0.10, PaceFast: 0.15},
}

// CalculateV1 implements daily_nutrition_target_v1 using the Mifflin–St Jeor
// resting-energy equation (https://pubmed.ncbi.nlm.nih.gov/2305711/). Inputs
// use kilograms, centimetres, completed years and kcal/day. It applies only to
// eligible adults aged 18–100 within the documented height and weight ranges.
// Energy rounds to the nearest 10 kcal and macros to the nearest gram, both
// half away from zero. Any change to the formula, coefficients, activity or
// goal policy, macro split, or rounding rule must create a new version rather
// than altering this function's historical results.
func CalculateV1(input CalculationInput) (CalculationResult, error) {
	eligibility := EvaluateEligibility(input)
	if !eligibility.Eligible {
		return CalculationResult{}, ErrAutomaticGoalNotEligible
	}
	adjustments, ok := goalAdjustments[input.Objective]
	if !ok {
		return CalculationResult{}, ErrInvalidGoal
	}
	adjustment, ok := adjustments[input.Pace]
	if !ok {
		return CalculationResult{}, ErrInvalidGoal
	}

	restingEnergy := 10*input.WeightKg + 6.25*input.HeightCm - 5*float64(eligibility.AgeYears)
	formula := "10 × 体重kg + 6.25 × 身高cm - 5 × 年龄 + 5"
	if input.BiologicalSex == SexMale {
		restingEnergy += 5
	} else {
		restingEnergy -= 161
		formula = "10 × 体重kg + 6.25 × 身高cm - 5 × 年龄 - 161"
	}
	activityMultiplier := activityMultipliers[input.ActivityLevel]
	maintenanceEnergy := restingEnergy * activityMultiplier
	goalEnergy := maintenanceEnergy * (1 + adjustment)
	energyTarget := int(roundTo(goalEnergy, 10))
	target := NutritionTarget{
		EnergyKcal:   energyTarget,
		ProteinGrams: int(roundTo(float64(energyTarget)*0.25/4, 1)),
		CarbGrams:    int(roundTo(float64(energyTarget)*0.50/4, 1)),
		FatGrams:     int(roundTo(float64(energyTarget)*0.25/9, 1)),
	}

	var pace *Pace
	if input.Pace != PaceNone {
		value := input.Pace
		pace = &value
	}
	goalLabel := map[Objective]string{
		ObjectiveFatLoss:    "应用减脂目标后的热量",
		ObjectiveMaintain:   "应用保持目标后的热量",
		ObjectiveMuscleGain: "应用增肌目标后的热量",
	}[input.Objective]
	calculation := GoalCalculation{
		CalculationVersion: CalculationVersionV1,
		Method: MethodDescription{
			ID:                EnergyMethodV1,
			DisplayName:       "Mifflin–St Jeor 静息能量估算",
			SourceURL:         "https://pubmed.ncbi.nlm.nih.gov/2305711/",
			FormulaExpression: formula,
		},
		MacroMethod: MacroMethodDescription{ID: MacroMethodV1, CarbPercent: 50, ProteinPercent: 25, FatPercent: 25},
		Inputs: CalculationInputs{
			BiologicalSex:         input.BiologicalSex,
			AgeYears:              eligibility.AgeYears,
			HeightCm:              input.HeightCm,
			WeightKg:              input.WeightKg,
			ActivityLevel:         input.ActivityLevel,
			ActivityMultiplier:    activityMultiplier,
			Objective:             input.Objective,
			Pace:                  pace,
			GoalAdjustmentPercent: int(adjustment * 100),
		},
		Steps: []CalculationStep{
			{Key: "restingEnergy", Label: "静息能量估算", Value: roundTo(restingEnergy, 0.1), Unit: "kcal/day"},
			{Key: "maintenanceEnergy", Label: "结合活动水平后的维持热量", Value: roundTo(maintenanceEnergy, 0.1), Unit: "kcal/day"},
			{Key: "goalEnergyBeforeRounding", Label: goalLabel, Value: roundTo(goalEnergy, 0.1), Unit: "kcal/day"},
			{Key: "finalEnergyTarget", Label: "取整后的每日热量目标", Value: float64(energyTarget), Unit: "kcal/day"},
		},
		Rounding:   RoundingDescription{Energy: "nearest_10_kcal", Macros: "nearest_1_gram", HalfRule: "half_away_from_zero"},
		Disclaimer: "该结果是基于统计公式的估算起点，不构成医学或营养处方。",
	}
	return CalculationResult{Target: target, Calculation: calculation, Warnings: []string{}}, nil
}
