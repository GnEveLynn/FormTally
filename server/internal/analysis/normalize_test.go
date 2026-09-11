package analysis

import "testing"

func TestNormalizeRoundsValuesAndBuildsPer100GramBasis(t *testing.T) {
	result, err := Normalize(Result{Items: []Item{{Name: " 米饭 ", Grams: 120.04, EnergyKcal: 155.6, ProteinGrams: 3.333, CarbGrams: 34.444, FatGrams: 0.222, Confidence: "medium"}}})
	if err != nil {
		t.Fatal(err)
	}
	item := result.Items[0]
	if item.Name != "米饭" || item.Grams != 120 || item.EnergyKcal != 156 || item.ProteinGrams != 3.3 || item.BasisPer100Grams.CarbGrams != 28.7 {
		t.Fatalf("normalized = %+v", item)
	}
}
