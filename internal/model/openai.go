package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/VemorPhose/TailChase/internal/project"
)

type OpenAICompatibleProvider struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewOpenAICompatibleProvider(cfg project.ModelConfig) (Provider, error) {
	if cfg.Provider != "openai_compatible" {
		return nil, fmt.Errorf("unsupported model.provider %q", cfg.Provider)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("model.base_url is required when prompt.mode is model")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, fmt.Errorf("model.model is required when prompt.mode is model")
	}
	if strings.TrimSpace(cfg.APIKeyEnv) == "" {
		return nil, fmt.Errorf("model.api_key_env is required when prompt.mode is model")
	}
	apiKey := strings.TrimSpace(os.Getenv(cfg.APIKeyEnv))
	if apiKey == "" {
		return nil, fmt.Errorf("model API key environment variable %s is not set", cfg.APIKeyEnv)
	}
	return OpenAICompatibleProvider{
		BaseURL: strings.TrimSpace(cfg.BaseURL),
		APIKey:  apiKey,
	}, nil
}

func (p OpenAICompatibleProvider) Generate(ctx context.Context, request Request) (Response, error) {
	if strings.TrimSpace(request.Model) == "" {
		return Response{}, fmt.Errorf("model request model is required")
	}
	if len(request.Messages) == 0 {
		return Response{}, fmt.Errorf("model request messages are required")
	}
	client := p.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	payload := openAIChatRequest{
		Model:    request.Model,
		Messages: make([]openAIMessage, 0, len(request.Messages)),
	}
	for _, message := range request.Messages {
		payload.Messages = append(payload.Messages, openAIMessage{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Response{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return Response{}, fmt.Errorf("model provider returned HTTP %d: %s", resp.StatusCode, message)
	}

	var decoded openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return Response{}, err
	}
	if len(decoded.Choices) == 0 {
		return Response{}, fmt.Errorf("model provider returned no choices")
	}
	content := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if content == "" {
		return Response{}, fmt.Errorf("model provider returned an empty prompt")
	}

	metadata := map[string]string{"provider": "openai_compatible"}
	if decoded.ID != "" {
		metadata["response_id"] = decoded.ID
	}
	return Response{Content: content, Metadata: metadata}, nil
}

type openAIChatRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}
