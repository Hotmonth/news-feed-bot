package summary

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiSummarizer struct {
	client  *genai.Client
	prompt  string
	enabled bool
	mu      sync.Mutex
}

func NewGeminiSummarizer(apiKey string, prompt string) *GeminiSummarizer {
	if apiKey == "" {
		log.Printf("Gemini summarizer disabled: no API key provided")
		return &GeminiSummarizer{
			enabled: false,
		}
	}
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Printf("[Error] failed to load the client: %v", err)
		return nil
	}

	s := &GeminiSummarizer{
		client: client,
		prompt: prompt,
	}

	log.Printf("Gemini summarizer enabled: %v", apiKey != "")

	if apiKey != "" {
		s.enabled = true
	}

	return s
}

func (s *GeminiSummarizer) Summarize(ctx context.Context, text string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled {
		return "", nil
	}
	model := s.client.GenerativeModel("gemini-pro")
	resp, err := model.GenerateContent(ctx, genai.Text(fmt.Sprintf("%s\n%s", text, s.prompt)))
	if err != nil {
		return "", err
	}
	return GeminiResponseToString(resp)
}

type Content struct {
	Parts []string `json:"Parts"`
	Role  string   `json:"Role"`
}

type Candidates struct {
	Content *Content `json:"Content"`
}

type ContentResponse struct {
	Candidates *[]Candidates `json:"Candidates"`
}

func GeminiResponseToString(resp *genai.GenerateContentResponse) (string, error) {
	marshalResponse, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshal JSON to structured object
	var generateResponse ContentResponse
	if err := json.Unmarshal(marshalResponse, &generateResponse); err != nil {
		log.Fatal(err)
	}

	// Build string from structured object
	var resultBuilder strings.Builder
	for _, cad := range *generateResponse.Candidates {
		if cad.Content != nil {
			for _, part := range cad.Content.Parts {
				resultBuilder.WriteString(part)
				resultBuilder.WriteString("\n") // Add new line after each part
			}
		}
	}

	// Get the built string
	result := resultBuilder.String()
	return result, nil
}
