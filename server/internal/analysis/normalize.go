package analysis

import (
	"fmt"
	"math"
	"strings"
)

func Normalize(result Result) (Result, error) {
	if err := validateResult(result); err != nil {
		return Result{}, err
	}
	for index := range result.Items {
		item := &result.Items[index]
		item.Name = strings.TrimSpace(item.Name)
		item.Grams = round1(item.Grams)
		item.EnergyKcal = math.Round(item.EnergyKcal)
		item.ProteinGrams, item.CarbGrams, item.FatGrams = round1(item.ProteinGrams), round1(item.CarbGrams), round1(item.FatGrams)
		if item.Grams <= 0 {
			return Result{}, fmt.Errorf("zero grams")
		}
		item.BasisPer100Grams = Nutrition{
			EnergyKcal: math.Round(item.EnergyKcal / item.Grams * 100), ProteinGrams: round1(item.ProteinGrams / item.Grams * 100),
			CarbGrams: round1(item.CarbGrams / item.Grams * 100), FatGrams: round1(item.FatGrams / item.Grams * 100),
		}
	}
	return result, nil
}

func round1(value float64) float64 { return math.Round(value*10) / 10 }
