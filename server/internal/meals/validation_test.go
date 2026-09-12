package meals

import (
	"errors"
	"testing"
	"time"
)

func TestUpdateRejectsItemIDFromAnotherMeal(t *testing.T) {
	foreign := "item_other"
	input := []ItemInput{{ID: &foreign, Name: "米饭", Grams: 100, Nutrition: Nutrition{EnergyKcal: 116}, Origin: "manual"}}
	if err := validateUpdateItemIDs([]Item{{ID: "item_owned"}}, input); !errors.Is(err, ErrValidation) {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateCreateRejectsInvalidItemsMealTypeAndFutureTime(t *testing.T) {
	now := time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC)
	valid := CreateInput{OccurredAt: "2026-09-11T07:00:00Z", MealType: "lunch", Items: []ItemInput{{Name: "米饭", Grams: 100, Nutrition: Nutrition{EnergyKcal: 116, ProteinGrams: 2.6, CarbGrams: 25.9, FatGrams: .3}}}}
	cases := []CreateInput{valid, valid, valid, valid}
	cases[0].Items = nil
	cases[1].Items[0].Grams = -1
	cases[2].MealType = "brunch"
	cases[3].OccurredAt = "2026-09-12T07:00:00Z"
	for _, input := range cases {
		if err := ValidateCreate(input, now); !errors.Is(err, ErrValidation) {
			t.Fatalf("invalid input accepted: %+v err=%v", input, err)
		}
	}
}
