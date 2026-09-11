package analysis

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

type Nutrition struct {
	EnergyKcal   float64 `json:"energyKcal"`
	ProteinGrams float64 `json:"proteinGrams"`
	CarbGrams    float64 `json:"carbGrams"`
	FatGrams     float64 `json:"fatGrams"`
}

type Item struct {
	Name             string    `json:"name"`
	Grams            float64   `json:"grams"`
	EnergyKcal       float64   `json:"energyKcal"`
	ProteinGrams     float64   `json:"proteinGrams"`
	CarbGrams        float64   `json:"carbGrams"`
	FatGrams         float64   `json:"fatGrams"`
	Confidence       string    `json:"confidence"`
	Assumption       *string   `json:"assumption"`
	BasisPer100Grams Nutrition `json:"basisPer100Grams,omitempty"`
}

type Result struct {
	Items      []Item  `json:"items"`
	Incomplete bool    `json:"incomplete"`
	Warning    *string `json:"warning"`
}

func ParseResult(reader io.Reader) (Result, error) {
	var result Result
	decoder := json.NewDecoder(io.LimitReader(reader, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return Result{}, fmt.Errorf("decode model output: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Result{}, fmt.Errorf("model output contains trailing data")
	}
	if err := validateResult(result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func validateResult(result Result) error {
	if len(result.Items) == 0 || len(result.Items) > 50 {
		return fmt.Errorf("model result must contain 1 to 50 items")
	}
	for index, item := range result.Items {
		name := strings.TrimSpace(item.Name)
		if name == "" || utf8.RuneCountInString(name) > 60 || item.Grams < .1 || item.Grams > 10000 || item.EnergyKcal < 0 || item.EnergyKcal > 10000 || item.ProteinGrams < 0 || item.ProteinGrams > 1000 || item.CarbGrams < 0 || item.CarbGrams > 1000 || item.FatGrams < 0 || item.FatGrams > 1000 {
			return fmt.Errorf("model item %d is outside supported ranges", index)
		}
		if item.Confidence != "high" && item.Confidence != "medium" && item.Confidence != "low" {
			return fmt.Errorf("model item %d has invalid confidence", index)
		}
		if item.Assumption != nil && utf8.RuneCountInString(*item.Assumption) > 200 {
			return fmt.Errorf("model item %d assumption is too long", index)
		}
	}
	if result.Warning != nil && utf8.RuneCountInString(*result.Warning) > 200 {
		return fmt.Errorf("model warning is too long")
	}
	return nil
}

func ResultJSONSchema() map[string]any {
	number := func(minimum, maximum float64) map[string]any {
		return map[string]any{"type": "number", "minimum": minimum, "maximum": maximum}
	}
	item := map[string]any{
		"type": "object", "additionalProperties": false,
		"required": []string{"name", "grams", "energyKcal", "proteinGrams", "carbGrams", "fatGrams", "confidence", "assumption"},
		"properties": map[string]any{
			"name": map[string]any{"type": "string", "minLength": 1, "maxLength": 60}, "grams": number(.1, 10000),
			"energyKcal": number(0, 10000), "proteinGrams": number(0, 1000), "carbGrams": number(0, 1000), "fatGrams": number(0, 1000),
			"confidence": map[string]any{"type": "string", "enum": []string{"high", "medium", "low"}},
			"assumption": map[string]any{"type": []string{"string", "null"}, "maxLength": 200},
		},
	}
	return map[string]any{
		"type": "object", "additionalProperties": false,
		"required": []string{"items", "incomplete", "warning"},
		"properties": map[string]any{
			"items":      map[string]any{"type": "array", "minItems": 1, "maxItems": 50, "items": item},
			"incomplete": map[string]any{"type": "boolean"},
			"warning":    map[string]any{"type": []string{"string", "null"}, "maxLength": 200},
		},
	}
}
