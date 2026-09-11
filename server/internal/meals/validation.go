package meals

import (
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrValidation = errors.New("meal validation failed")

func ValidateCreate(input CreateInput, now time.Time) error {
	occurred, err := time.Parse(time.RFC3339, input.OccurredAt)
	if err != nil || occurred.After(now) {
		return ErrValidation
	}
	if input.MealType != "breakfast" && input.MealType != "lunch" && input.MealType != "dinner" && input.MealType != "snack" || len(input.Items) == 0 || len(input.Items) > 50 {
		return ErrValidation
	}
	for _, item := range input.Items {
		if strings.TrimSpace(item.Name) == "" || utf8.RuneCountInString(strings.TrimSpace(item.Name)) > 60 || !finiteRange(item.Grams, .1, 10000) || item.Nutrition.EnergyKcal < 0 || item.Nutrition.EnergyKcal > 10000 || !finiteRange(item.Nutrition.ProteinGrams, 0, 1000) || !finiteRange(item.Nutrition.CarbGrams, 0, 1000) || !finiteRange(item.Nutrition.FatGrams, 0, 1000) {
			return ErrValidation
		}
	}
	return nil
}
func finiteRange(value, min, max float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= min && value <= max
}
