package analysis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Gokul-Eswar/Spectre/internal/analyzer"
	"github.com/spf13/viper"
)

type OllamaModel struct {
	Name string `json:"name"`
}

type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

type OpenAIModelItem struct {
	ID string `json:"id"`
}

type OpenAIModelsResponse struct {
	Data []OpenAIModelItem `json:"data"`
}

// GetSupportedProviders returns the list of all supported LLM providers.
func GetSupportedProviders() []string {
	return []string{
		"ollama",
		"openai",
		"anthropic",
		"gemini",
		"groq",
		"openrouter",
		"lmstudio",
		"localai",
		"custom",
	}
}

// GetPresetsForProvider returns standard URL, starter model, and known models for a provider.
func GetPresetsForProvider(provider string) (string, string, []string) {
	p := strings.ToLower(strings.TrimSpace(provider))
	switch p {
	case "openai":
		return "https://api.openai.com/v1", "gpt-4o-mini", []string{
			"gpt-4o-mini",
			"gpt-4o",
			"gpt-4-turbo",
			"gpt-3.5-turbo",
			"o1",
			"o3-mini",
		}
	case "anthropic":
		return "https://api.anthropic.com/v1", "claude-3-5-sonnet-20241022", []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
			"claude-3-haiku-20240307",
		}
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta/openai", "gemini-1.5-flash", []string{
			"gemini-1.5-flash",
			"gemini-1.5-pro",
			"gemini-2.0-flash",
			"gemini-1.5-flash-8b",
		}
	case "groq":
		return "https://api.groq.com/openai/v1", "llama-3.3-70b-versatile", []string{
			"llama-3.3-70b-versatile",
			"llama-3.1-8b-instant",
			"mixtral-8x7b-32768",
			"deepseek-r1-distill-llama-70b",
		}
	case "openrouter":
		return "https://openrouter.ai/api/v1", "meta-llama/llama-3.3-70b-instruct", []string{
			"meta-llama/llama-3.3-70b-instruct",
			"anthropic/claude-3.5-sonnet",
			"google/gemini-flash-1.5",
			"openai/gpt-4o-mini",
			"deepseek/deepseek-r1",
		}
	case "lmstudio":
		return "http://localhost:1234/v1", "local-model", []string{
			"local-model",
			"llama-3.2-3b-instruct",
			"qwen2.5-7b-instruct",
			"mistral-7b-instruct",
		}
	case "localai":
		return "http://localhost:8080/v1", "gpt-4", []string{
			"gpt-4",
			"llama3",
			"mistral",
		}
	case "ollama":
		return "http://localhost:11434", "llama3", []string{
			"llama3",
			"llama3:8b",
			"mistral",
			"qwen2.5:7b",
			"phi3",
			"llava",
		}
	default:
		return "http://localhost:11434", "llama3", []string{"llama3"}
	}
}

// FetchAvailableModels connects to the configured LLM provider and retrieves
// available models, with graceful fallback to curated presets.
func FetchAvailableModels() ([]string, error) {
	cfg := analyzer.GetLLMConfig()
	provider := strings.ToLower(cfg.Provider)

	currentModel := viper.GetString("llm.model")
	if currentModel == "" {
		currentModel = analyzer.GetDefaultModelForProvider(provider)
	}

	var discovered []string

	if provider == "ollama" {
		models, err := fetchOllamaModels(cfg.URL)
		if err == nil && len(models) > 0 {
			discovered = models
		}
	} else if provider == "openai" || provider == "groq" || provider == "openrouter" || provider == "lmstudio" || provider == "localai" || provider == "custom" {
		models, err := fetchOpenAICompatibleModels(cfg.URL, cfg.APIKey)
		if err == nil && len(models) > 0 {
			discovered = models
		}
	}

	// If dynamic discovery returned nothing or provider doesn't support it, use provider presets
	if len(discovered) == 0 {
		_, _, presetModels := GetPresetsForProvider(provider)
		discovered = append(discovered, presetModels...)
	}

	// Ensure currently configured model is present
	found := false
	for _, m := range discovered {
		if strings.EqualFold(m, currentModel) {
			found = true
			break
		}
	}
	if !found && currentModel != "" {
		discovered = append([]string{currentModel}, discovered...)
	}

	return discovered, nil
}

func fetchOllamaModels(rawURL string) ([]string, error) {
	baseURL := strings.TrimRight(rawURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/api/generate")
	baseURL = strings.TrimSuffix(baseURL, "/api/chat")
	tagsURL := baseURL + "/api/tags"

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(tagsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama at %s: %w", tagsURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ollama API returned status: %s", resp.Status)
	}

	var tagsResp OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w", err)
	}

	var models []string
	for _, m := range tagsResp.Models {
		models = append(models, m.Name)
	}
	return models, nil
}

func fetchOpenAICompatibleModels(rawURL, apiKey string) ([]string, error) {
	baseURL := strings.TrimRight(rawURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/chat/completions")
	baseURL = strings.TrimSuffix(baseURL, "/completions")
	modelsURL := baseURL + "/models"

	client := http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", modelsURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("models API returned status: %s", resp.Status)
	}

	var modelsResp OpenAIModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	var models []string
	for _, m := range modelsResp.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}
	return models, nil
}

// TestLLMConnection verifies connectivity and credentials with the active LLM backend.
func TestLLMConnection() (bool, time.Duration, string, error) {
	cfg := analyzer.GetLLMConfig()
	provider := strings.ToLower(cfg.Provider)
	start := time.Now()

	client := http.Client{Timeout: 5 * time.Second}

	var testURL string
	var req *http.Request
	var err error

	switch provider {
	case "ollama":
		baseURL := strings.TrimRight(cfg.URL, "/")
		baseURL = strings.TrimSuffix(baseURL, "/api/generate")
		baseURL = strings.TrimSuffix(baseURL, "/api/chat")
		testURL = baseURL + "/api/tags"
		req, err = http.NewRequest("GET", testURL, nil)

	case "anthropic":
		baseURL := strings.TrimRight(cfg.URL, "/")
		baseURL = strings.TrimSuffix(baseURL, "/v1")
		testURL = baseURL + "/v1/models"
		req, err = http.NewRequest("GET", testURL, nil)
		if err == nil {
			req.Header.Set("x-api-key", cfg.APIKey)
			req.Header.Set("anthropic-version", "2023-06-01")
		}

	case "gemini":
		baseURL := strings.TrimRight(cfg.URL, "/")
		baseURL = strings.TrimSuffix(baseURL, "/chat/completions")
		testURL = baseURL + "/models"
		req, err = http.NewRequest("GET", testURL, nil)
		if err == nil && cfg.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
		}

	default: // openai, groq, openrouter, lmstudio, localai, custom
		baseURL := strings.TrimRight(cfg.URL, "/")
		baseURL = strings.TrimSuffix(baseURL, "/chat/completions")
		baseURL = strings.TrimSuffix(baseURL, "/completions")
		testURL = baseURL + "/models"
		req, err = http.NewRequest("GET", testURL, nil)
		if err == nil && cfg.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
		}
	}

	if err != nil {
		return false, 0, "", err
	}

	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return false, latency, fmt.Sprintf("Connection failed (%s)", provider), err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, latency, fmt.Sprintf("Connected to %s (%dms)", provider, latency.Milliseconds()), nil
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return false, latency, fmt.Sprintf("Auth failed (%d %s) - Check API Key", resp.StatusCode, resp.Status), fmt.Errorf("unauthorized")
	}

	// For Anthropic or providers that return 400 on GET, if server is reachable, report reachability
	if resp.StatusCode < 500 {
		return true, latency, fmt.Sprintf("Endpoint reached (%s, HTTP %d)", provider, resp.StatusCode), nil
	}

	return false, latency, fmt.Sprintf("Server error from %s (HTTP %d)", provider, resp.StatusCode), fmt.Errorf("status %d", resp.StatusCode)
}
