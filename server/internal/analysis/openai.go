package analysis

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

type OpenAIConfig struct {
	APIKey, Model, BaseURL string
	Timeout                time.Duration
	HTTPClient             *http.Client
}

type OpenAIAnalyzer struct {
	client  openai.Client
	model   string
	timeout time.Duration
}

func NewOpenAIAnalyzer(config OpenAIConfig) *OpenAIAnalyzer {
	options := []option.RequestOption{option.WithAPIKey(config.APIKey)}
	if config.BaseURL != "" {
		options = append(options, option.WithBaseURL(config.BaseURL))
	}
	if config.HTTPClient != nil {
		options = append(options, option.WithHTTPClient(config.HTTPClient))
	}
	return &OpenAIAnalyzer{client: openai.NewClient(options...), model: config.Model, timeout: config.Timeout}
}

func (a *OpenAIAnalyzer) Analyze(ctx context.Context, image []byte) (Result, Metadata, error) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	imageInput := responses.ResponseInputContentParamOfInputImage(responses.ResponseInputImageDetailHigh)
	imageInput.OfInputImage.ImageURL = openai.String("data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(image))
	content := responses.ResponseInputMessageContentListParam{responses.ResponseInputContentParamOfInputText(prompt + "\n提示版本：" + PromptVersion), imageInput}
	response, err := a.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: a.model,
		Input: responses.ResponseNewParamsInputUnion{OfInputItemList: responses.ResponseInputParam{
			responses.ResponseInputItemParamOfMessage(content, responses.EasyInputMessageRoleUser),
		}},
		Text:  responses.ResponseTextConfigParam{Format: responses.ResponseFormatTextConfigParamOfJSONSchema("meal_analysis", ResultJSONSchema())},
		Store: openai.Bool(false),
	})
	meta := Metadata{Model: a.model, PromptVersion: PromptVersion, Duration: time.Since(started)}
	if err != nil {
		code := "AI_UNAVAILABLE"
		if ctx.Err() != nil {
			code = "AI_TIMEOUT"
		}
		return Result{}, meta, &Failure{Code: code, Retryable: true, Err: err}
	}
	meta.ResponseStatus = string(response.Status)
	parsed, err := ParseResult(bytes.NewBufferString(response.OutputText()))
	if err != nil {
		return Result{}, meta, &Failure{Code: "AI_OUTPUT_INVALID", Retryable: true, Err: err}
	}
	normalized, err := Normalize(parsed)
	if err != nil {
		return Result{}, meta, &Failure{Code: "AI_OUTPUT_INVALID", Retryable: true, Err: err}
	}
	if len(normalized.Items) == 0 {
		return Result{}, meta, &Failure{Code: "FOOD_NOT_RECOGNIZED", Retryable: false, Err: fmt.Errorf("no food recognized")}
	}
	return normalized, meta, nil
}
