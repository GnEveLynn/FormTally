package analysis

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLogAnalysisResultCorrelatesAIOutcomeWithoutImageContent(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	logAnalysisResult(logger, "request-123", View{
		ID: "analysis_123", Status: "review_required", ProcessingMode: "ai",
		Items: []ItemView{{Name: "米饭"}}, Metadata: Metadata{Model: "qwen-vl", PromptVersion: "meal-v1", ResponseStatus: "stop", Duration: 9 * time.Second},
	})
	logText := output.String()
	for _, value := range []string{"meal analysis completed", "request-123", "analysis_123", "review_required", "qwen-vl", "meal-v1", "stop", `"item_count":1`, `"ai_duration_ms":9000`} {
		if !strings.Contains(logText, value) {
			t.Fatalf("analysis log missing %q: %s", value, logText)
		}
	}
	if strings.Contains(logText, "米饭") {
		t.Fatalf("analysis log leaked model/image-derived content: %s", logText)
	}
}
