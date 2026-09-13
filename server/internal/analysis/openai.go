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
	APIStyle               string
	Timeout                time.Duration
	HTTPClient             *http.Client
}

type OpenAIAnalyzer struct {
	client   openai.Client
	model    string
	apiStyle string
	timeout  time.Duration
}

func NewOpenAIAnalyzer(config OpenAIConfig) *OpenAIAnalyzer {
	options := []option.RequestOption{option.WithAPIKey(config.APIKey)}
	if config.BaseURL != "" {
		options = append(options, option.WithBaseURL(config.BaseURL))
	}
	if config.HTTPClient != nil {
		options = append(options, option.WithHTTPClient(config.HTTPClient))
	}
	if config.APIStyle == "" {
		config.APIStyle = "responses"
	}
	return &OpenAIAnalyzer{client: openai.NewClient(options...), model: config.Model, apiStyle: config.APIStyle, timeout: config.Timeout}
}

func (a *OpenAIAnalyzer) Analyze(ctx context.Context, image []byte) (Result, Metadata, error) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	dataURL := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(image)
	var output, responseStatus string
	var err error
	if a.apiStyle == "chat_completions" {
		output, responseStatus, err = a.analyzeChatCompletions(ctx, dataURL)
	} else {
		output, responseStatus, err = a.analyzeResponses(ctx, dataURL)
	}
	meta := Metadata{Model: a.model, PromptVersion: PromptVersion, Duration: time.Since(started)}
	if err != nil {
		code := "AI_UNAVAILABLE"
		if ctx.Err() != nil {
			code = "AI_TIMEOUT"
		}
		return Result{}, meta, &Failure{Code: code, Retryable: true, Err: err}
	}
	meta.ResponseStatus = responseStatus
	parsed, err := ParseResult(bytes.NewBufferString(output))
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

func (a *OpenAIAnalyzer) analyzeResponses(ctx context.Context, dataURL string) (string, string, error) {
	imageInput := responses.ResponseInputContentParamOfInputImage(responses.ResponseInputImageDetailHigh)
	imageInput.OfInputImage.ImageURL = openai.String(dataURL)
	content := responses.ResponseInputMessageContentListParam{responses.ResponseInputContentParamOfInputText(prompt + "\n提示版本：" + PromptVersion), imageInput}
	response, err := a.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: a.model,
		Input: responses.ResponseNewParamsInputUnion{OfInputItemList: responses.ResponseInputParam{
			responses.ResponseInputItemParamOfMessage(content, responses.EasyInputMessageRoleUser),
		}},
		Text:  responses.ResponseTextConfigParam{Format: responses.ResponseFormatTextConfigParamOfJSONSchema("meal_analysis", ResultJSONSchema())},
		Store: openai.Bool(false),
	})
	if err != nil {
		return "", "", err
	}
	return response.OutputText(), string(response.Status), nil
}

func (a *OpenAIAnalyzer) analyzeChatCompletions(ctx context.Context, dataURL string) (string, string, error) {
	response, err := a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: a.model,
		Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
			openai.TextContentPart(prompt + "\n提示版本：" + PromptVersion),
			openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{URL: dataURL, Detail: "high"}),
		})},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
			JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{Name: "meal_analysis", Schema: ResultJSONSchema(), Strict: openai.Bool(true)},
		}},
		Store: openai.Bool(false),
	}, option.WithJSONSet("enable_thinking", false))
	if err != nil {
		return "", "", err
	}
	if len(response.Choices) == 0 {
		return "", "", fmt.Errorf("chat completion returned no choices")
	}
	return response.Choices[0].Message.Content, response.Choices[0].FinishReason, nil
}
