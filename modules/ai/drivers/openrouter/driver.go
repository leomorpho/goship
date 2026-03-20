package openrouterdriver

import (
	"net/http"
	"strings"

	"github.com/leomorpho/goship/modules/ai"
	openaidriver "github.com/leomorpho/goship/modules/ai/drivers/openai"
	openai "github.com/sashabaranov/go-openai"
)

type OpenRouterDriver struct {
	*openaidriver.OpenAIDriver
}

func New(apiKey string, defaultModel string, siteURL string, siteName string) *OpenRouterDriver {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://openrouter.ai/api/v1"
	config.HTTPClient = &headerClient{
		base: http.DefaultClient,
		headers: map[string]string{
			"HTTP-Referer": strings.TrimSpace(siteURL),
			"X-Title":      strings.TrimSpace(siteName),
		},
	}

	if strings.TrimSpace(defaultModel) == "" {
		defaultModel = ai.ORClaudeHaiku4
	}

	return &OpenRouterDriver{
		OpenAIDriver: openaidriver.NewWithConfig(config, defaultModel),
	}
}

func NewWithConfig(config openai.ClientConfig, defaultModel string) *OpenRouterDriver {
	if strings.TrimSpace(defaultModel) == "" {
		defaultModel = ai.ORClaudeHaiku4
	}

	return &OpenRouterDriver{
		OpenAIDriver: openaidriver.NewWithConfig(config, defaultModel),
	}
}

type headerClient struct {
	base    openai.HTTPDoer
	headers map[string]string
}

func (c *headerClient) Do(req *http.Request) (*http.Response, error) {
	for key, value := range c.headers {
		if strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	return c.base.Do(req)
}
