package meals

import "time"

type Nutrition struct {
	EnergyKcal   int     `json:"energyKcal"`
	ProteinGrams float64 `json:"proteinGrams"`
	CarbGrams    float64 `json:"carbGrams"`
	FatGrams     float64 `json:"fatGrams"`
}
type ItemInput struct {
	ID               *string    `json:"id,omitempty"`
	DraftItemID      *string    `json:"draftItemId"`
	Name             string     `json:"name"`
	Grams            float64    `json:"grams"`
	Nutrition        Nutrition  `json:"nutrition"`
	BasisPer100Grams *Nutrition `json:"basisPer100Grams"`
	Origin           string     `json:"origin"`
	Confidence       *string    `json:"confidence"`
	Assumption       *string    `json:"assumption"`
}
type Item struct {
	ID string `json:"id"`
	ItemInput
}
type CreateInput struct {
	AnalysisID *string     `json:"analysisId"`
	OccurredAt string      `json:"occurredAt"`
	MealType   string      `json:"mealType"`
	Items      []ItemInput `json:"items"`
}
type Meal struct {
	ID, UserID, LocalDate string
	OccurredAt            time.Time
	MealType              string
	ImageKey              *string
	Items                 []Item
	Totals                Nutrition
	Revision              int
	CreatedAt, UpdatedAt  time.Time
}
type ImageView struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	MimeType  string    `json:"mimeType"`
}
type MealView struct {
	ID             string     `json:"id"`
	OccurredAt     time.Time  `json:"occurredAt"`
	LocalDate      string     `json:"localDate"`
	MealType       string     `json:"mealType"`
	Image          *ImageView `json:"image"`
	Items          []Item     `json:"items"`
	Totals         Nutrition  `json:"totals"`
	EstimateNotice string     `json:"estimateNotice"`
	Revision       int        `json:"revision"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}
type CreateResult struct {
	Meal               MealView `json:"meal"`
	AffectedLocalDates []string `json:"affectedLocalDates"`
}
type UpdateInput struct {
	ExpectedRevision int          `json:"expectedRevision"`
	OccurredAt       *string      `json:"occurredAt"`
	MealType         *string      `json:"mealType"`
	Items            *[]ItemInput `json:"items"`
}
type DeleteResult struct {
	DeletedMealID      string   `json:"deletedMealId"`
	AffectedLocalDates []string `json:"affectedLocalDates"`
}
