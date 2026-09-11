package meals

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNutritionFixtures(t *testing.T) {
	data, err := os.ReadFile("../../../packages/domain/testdata/meal-editor-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name  string    `json:"name"`
		Basis Nutrition `json:"basis"`
		Grams float64   `json:"grams"`
		Want  Nutrition `json:"want"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			if got := Scale(test.Basis, test.Grams); got != test.Want {
				t.Fatalf("Scale()=%+v want %+v", got, test.Want)
			}
		})
	}
}

func TestTotalsUseFinalEditedValues(t *testing.T) {
	got := Sum([]Item{{Nutrition: Nutrition{EnergyKcal: 198, ProteinGrams: 37.2, FatGrams: 4.3}}, {Nutrition: Nutrition{EnergyKcal: 232, ProteinGrams: 5.2, CarbGrams: 51.8, FatGrams: .6}}})
	want := Nutrition{EnergyKcal: 430, ProteinGrams: 42.4, CarbGrams: 51.8, FatGrams: 4.9}
	if got != want {
		t.Fatalf("Sum()=%+v want %+v", got, want)
	}
}
