package meals

import "math"

func Scale(basis Nutrition, grams float64) Nutrition {
	return Nutrition{EnergyKcal: int(math.Round(float64(basis.EnergyKcal) * grams / 100)), ProteinGrams: round1(float64(basis.ProteinGrams) * grams / 100), CarbGrams: round1(basis.CarbGrams * grams / 100), FatGrams: round1(basis.FatGrams * grams / 100)}
}
func Sum(items []Item) Nutrition {
	var total Nutrition
	for _, item := range items {
		total.EnergyKcal += item.Nutrition.EnergyKcal
		total.ProteinGrams += item.Nutrition.ProteinGrams
		total.CarbGrams += item.Nutrition.CarbGrams
		total.FatGrams += item.Nutrition.FatGrams
	}
	total.ProteinGrams = round1(total.ProteinGrams)
	total.CarbGrams = round1(total.CarbGrams)
	total.FatGrams = round1(total.FatGrams)
	return total
}
func round1(value float64) float64 { return math.Round(value*10) / 10 }
