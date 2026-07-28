package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type GeminiService interface {
	TransliterateEnglishToHindi(ctx context.Context, text string) (string, error)
}

type geminiService struct {
	apiKey string
	model  string
	client *http.Client
	cache  sync.Map // Map[string]string
}

func NewGeminiService(apiKey, model string) GeminiService {
	if model == "" {
		model = "gemini-3.5-flash-lite"
	}
	return &geminiService{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 300 * time.Millisecond},
	}
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

var englishCharRegex = regexp.MustCompile(`[a-zA-Z]`)

func (s *geminiService) TransliterateEnglishToHindi(ctx context.Context, text string) (string, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || s.apiKey == "" {
		return "", nil
	}

	// Only transliterate if input contains English alphabet characters
	if !englishCharRegex.MatchString(trimmed) {
		return "", nil
	}

	cacheKey := strings.ToLower(trimmed)
	if cachedVal, ok := s.cache.Load(cacheKey); ok {
		return cachedVal.(string), nil
	}

	// Use context with 250ms timeout for ultra-fast execution without blocking searches
	reqCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()

	prompt := fmt.Sprintf("Transliterate the Indian name or text '%s' into Devanagari Hindi script. Return ONLY the Hindi text without any quotes, punctuation, explanation or extra text.", trimmed)

	reqPayload := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal gemini request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", s.model, s.apiKey)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create gemini HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("Gemini API call timed out or failed: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read gemini response: %w", err)
	}

	var gResp geminiResponse
	if err := json.Unmarshal(respBytes, &gResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal gemini response: %w", err)
	}

	if gResp.Error != nil {
		log.Printf("Gemini API returned error: %s (code %d)", gResp.Error.Message, gResp.Error.Code)
		return "", fmt.Errorf("gemini api error: %s", gResp.Error.Message)
	}

	if len(gResp.Candidates) == 0 || len(gResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty candidates in gemini response")
	}

	hindiResult := strings.TrimSpace(gResp.Candidates[0].Content.Parts[0].Text)
	// Remove quotes or stray newlines
	hindiResult = strings.Trim(hindiResult, "\"'`\n\r\t")

	if hindiResult != "" {
		s.cache.Store(cacheKey, hindiResult)
		log.Printf("✨ Gemini Transliterated: English '%s' -> Hindi '%s'", trimmed, hindiResult)
	}

	return hindiResult, nil
}
