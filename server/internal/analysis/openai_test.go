package analysis

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOpenAIAnalyzerSendsImageSchemaPromptVersionAndDoesNotStoreResponse(t *testing.T) {
	var requestBody string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		requestBody = string(body)
		response := `{"id":"resp_1","object":"response","created_at":1,"status":"completed","model":"test-model","output":[{"id":"msg_1","type":"message","status":"completed","role":"assistant","content":[{"type":"output_text","text":"{\"items\":[{\"name\":\"米饭\",\"grams\":100,\"energyKcal\":116,\"proteinGrams\":2.6,\"carbGrams\":25.9,\"fatGrams\":0.3,\"confidence\":\"high\",\"assumption\":null}],\"incomplete\":false,\"warning\":null}","annotations":[]}]}]}`
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(response)), Request: r}, nil
	})}
	analyzer := NewOpenAIAnalyzer(OpenAIConfig{APIKey: "test-key", Model: "test-model", Timeout: time.Second, BaseURL: "https://api.test/v1", HTTPClient: client})
	result, meta, err := analyzer.Analyze(context.Background(), []byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || meta.Model != "test-model" || meta.PromptVersion != PromptVersion {
		t.Fatalf("result=%+v meta=%+v", result, meta)
	}
	for _, required := range []string{"data:image/jpeg;base64,AQID", `"type":"json_schema"`, PromptVersion, `"store":false`} {
		if !bytes.Contains([]byte(requestBody), []byte(required)) {
			t.Fatalf("request missing %q: %s", required, requestBody)
		}
	}
}

func TestOpenAILive(t *testing.T) {
	if os.Getenv("FORMTALLY_OPENAI_INTEGRATION") != "1" {
		t.Skip("set FORMTALLY_OPENAI_INTEGRATION=1 to run the live OpenAI check")
	}
	key, model := os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_MODEL")
	if key == "" || model == "" {
		t.Fatal("OPENAI_API_KEY and OPENAI_MODEL are required for the live check")
	}
	var imageBytes bytes.Buffer
	meal := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			meal.Set(x, y, color.RGBA{R: 235, G: 190, B: 90, A: 255})
		}
	}
	if err := jpeg.Encode(&imageBytes, meal, nil); err != nil {
		t.Fatal(err)
	}
	result, _, err := NewOpenAIAnalyzer(OpenAIConfig{APIKey: key, Model: model, Timeout: 30 * time.Second}).Analyze(context.Background(), imageBytes.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) == 0 {
		t.Fatal("live analyzer returned no food items")
	}
}
