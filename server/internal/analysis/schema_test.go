package analysis

import (
	"strings"
	"testing"
)

func TestParseResultAcceptsPartialAndLowConfidenceItems(t *testing.T) {
	result, err := ParseResult(strings.NewReader(`{
		"items":[{"name":"鸡胸肉","grams":120,"energyKcal":198,"proteinGrams":37.2,"carbGrams":0,"fatGrams":4.3,"confidence":"low","assumption":"按熟制估算"}],
		"incomplete":true,"warning":"可能还有未识别食物"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || !result.Incomplete || result.Items[0].Confidence != "low" {
		t.Fatalf("result = %+v", result)
	}
}

func TestParseResultRejectsUntrustedModelOutput(t *testing.T) {
	cases := []string{
		`not json`,
		`{"items":[],"incomplete":false,"warning":null}`,
		`{"items":[{"name":"米饭","grams":-1,"energyKcal":1,"proteinGrams":1,"carbGrams":1,"fatGrams":1,"confidence":"high","assumption":null}],"incomplete":false,"warning":null}`,
		`{"items":[{"name":"米饭","grams":100,"energyKcal":99999,"proteinGrams":1,"carbGrams":1,"fatGrams":1,"confidence":"certain","assumption":null}],"incomplete":false,"warning":null}`,
		`{"items":[{"name":"米饭","grams":100,"energyKcal":100,"proteinGrams":1,"carbGrams":1,"fatGrams":1,"confidence":"high","assumption":null,"extra":1}],"incomplete":false,"warning":null}`,
	}
	for _, input := range cases {
		if _, err := ParseResult(strings.NewReader(input)); err == nil {
			t.Fatalf("invalid model output accepted: %s", input)
		}
	}
}

func TestParseResultRejectsUserFacingTextWithoutChinese(t *testing.T) {
	cases := []string{
		`{"items":[{"name":"Fried Chicken Burger","grams":280,"energyKcal":650,"proteinGrams":28,"carbGrams":45,"fatGrams":32,"confidence":"high","assumption":null}],"incomplete":false,"warning":null}`,
		`{"items":[{"name":"炸鸡汉堡","grams":280,"energyKcal":650,"proteinGrams":28,"carbGrams":45,"fatGrams":32,"confidence":"low","assumption":"Typical serving"}],"incomplete":false,"warning":null}`,
		`{"items":[{"name":"炸鸡汉堡","grams":280,"energyKcal":650,"proteinGrams":28,"carbGrams":45,"fatGrams":32,"confidence":"high","assumption":null}],"incomplete":true,"warning":"Image is blurry"}`,
	}
	for _, input := range cases {
		if _, err := ParseResult(strings.NewReader(input)); err == nil {
			t.Fatalf("model output without Chinese accepted: %s", input)
		}
	}
}

func TestResultSchemaIsStrictAtEveryObjectLevel(t *testing.T) {
	schema := ResultJSONSchema()
	if schema["additionalProperties"] != false {
		t.Fatal("root schema allows additional properties")
	}
	items := schema["properties"].(map[string]any)["items"].(map[string]any)
	if items["items"].(map[string]any)["additionalProperties"] != false {
		t.Fatal("item schema allows additional properties")
	}
}
