package profile

import (
	"errors"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/goals"
)

var (
	ErrValidation       = errors.New("profile validation failed")
	ErrRevisionConflict = errors.New("profile revision conflict")
)

type PutInput struct {
	BiologicalSex    goals.BiologicalSex `json:"biologicalSex"`
	BirthDate        string              `json:"birthDate"`
	HeightCm         float64             `json:"heightCm"`
	WeightKg         float64             `json:"weightKg"`
	ActivityLevel    goals.ActivityLevel `json:"activityLevel"`
	Timezone         string              `json:"timezone"`
	HealthContext    goals.HealthContext `json:"healthContext"`
	ExpectedRevision *int                `json:"expectedRevision,omitempty"`
}

type View struct {
	BiologicalSex         goals.BiologicalSex `json:"biologicalSex"`
	BirthDate             string              `json:"birthDate"`
	HeightCm              float64             `json:"heightCm"`
	WeightKg              float64             `json:"weightKg"`
	ActivityLevel         goals.ActivityLevel `json:"activityLevel"`
	Timezone              string              `json:"timezone"`
	HealthContext         goals.HealthContext `json:"healthContext"`
	AutomaticGoalEligible bool                `json:"automaticGoalEligible"`
	Revision              int                 `json:"revision"`
	UpdatedAt             time.Time           `json:"updatedAt"`
}
