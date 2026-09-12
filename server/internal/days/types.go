package days

import (
	"github.com/GnEveLynn/FormTally/server/internal/goals"
	"github.com/GnEveLynn/FormTally/server/internal/meals"
	"time"
)

type MetricProgress struct {
	Consumed  float64 `json:"consumed"`
	Target    float64 `json:"target"`
	Remaining float64 `json:"remaining"`
	OverBy    float64 `json:"overBy"`
	Percent   float64 `json:"percent"`
	Status    string  `json:"status"`
}
type Progress struct {
	Energy  MetricProgress `json:"energy"`
	Protein MetricProgress `json:"protein"`
	Carb    MetricProgress `json:"carb"`
	Fat     MetricProgress `json:"fat"`
}
type MealSummary struct {
	ID         string          `json:"id"`
	OccurredAt time.Time       `json:"occurredAt"`
	Totals     meals.Nutrition `json:"totals"`
}
type MealGroup struct {
	MealType string        `json:"mealType"`
	Meals    []MealSummary `json:"meals"`
}
type View struct {
	LocalDate  string                 `json:"localDate"`
	Target     *goals.NutritionTarget `json:"target"`
	Totals     meals.Nutrition        `json:"totals"`
	Progress   *Progress              `json:"progress"`
	MealGroups []MealGroup            `json:"mealGroups"`
}
type HistoryDay struct {
	LocalDate  string `json:"localDate"`
	MealCount  int    `json:"mealCount"`
	EnergyKcal int    `json:"energyKcal"`
	Status     string `json:"status"`
}
type HistoryView struct {
	Month    string       `json:"month"`
	Timezone string       `json:"timezone"`
	Days     []HistoryDay `json:"days"`
}
