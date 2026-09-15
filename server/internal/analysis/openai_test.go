package analysis

import (
	"bytes"
	"context"
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

func TestOpenAIAnalyzerUsesChatCompletionsForQwenStructuredOutput(t *testing.T) {
	var requestPath, requestBody string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		requestPath, requestBody = r.URL.Path, string(body)
		response := `{"id":"chatcmpl_1","object":"chat.completion","created":1,"model":"test-model","choices":[{"index":0,"message":{"role":"assistant","content":"{\"items\":[{\"name\":\"米饭\",\"grams\":100,\"energyKcal\":116,\"proteinGrams\":2.6,\"carbGrams\":25.9,\"fatGrams\":0.3,\"confidence\":\"high\",\"assumption\":null}],\"incomplete\":false,\"warning\":null}"},"finish_reason":"stop"}]}`
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(response)), Request: r}, nil
	})}
	analyzer := NewOpenAIAnalyzer(OpenAIConfig{APIKey: "test-key", Model: "test-model", APIStyle: "chat_completions", Timeout: time.Second, BaseURL: "https://api.test/v1", HTTPClient: client})
	result, meta, err := analyzer.Analyze(context.Background(), []byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	if requestPath != "/v1/chat/completions" {
		t.Fatalf("request path = %q", requestPath)
	}
	if len(result.Items) != 1 || meta.ResponseStatus != "stop" {
		t.Fatalf("result=%+v meta=%+v", result, meta)
	}
	for _, required := range []string{"data:image/jpeg;base64,AQID", `"type":"json_schema"`, PromptVersion, "所有面向用户的文本必须使用简体中文", `"store":false`, `"enable_thinking":false`} {
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
	imagePath := os.Getenv("FORMTALLY_OPENAI_TEST_IMAGE")
	if imagePath == "" {
		t.Fatal("FORMTALLY_OPENAI_TEST_IMAGE must point to a real meal image")
	}
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := NewOpenAIAnalyzer(OpenAIConfig{APIKey: key, Model: model, APIStyle: os.Getenv("OPENAI_API_STYLE"), BaseURL: os.Getenv("OPENAI_BASE_URL"), Timeout: 30 * time.Second}).Analyze(context.Background(), imageBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) == 0 {
		t.Fatal("live analyzer returned no food items")
	}
}
