package service_test

import (
	"context"
	"testing"
	"time"

	"durg-voter-api/pkg/config"
	"durg-voter-api/pkg/service"
)

func TestGeminiTransliteration(t *testing.T) {
	cfg := config.LoadConfig()
	if cfg.GeminiAPIKey == "" {
		t.Skip("Skipping Gemini API test because GEMINI_API_KEY is empty")
	}

	geminiSvc := service.NewGeminiService(cfg.GeminiAPIKey, cfg.GeminiModel)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	englishName := "Surendra Kumar"
	hindiName, err := geminiSvc.TransliterateEnglishToHindi(ctx, englishName)
	if err != nil {
		t.Fatalf("Transliteration failed with error: %v", err)
	}

	if hindiName == "" {
		t.Fatalf("Expected non-empty Hindi transliteration for '%s'", englishName)
	}

	t.Logf("✨ Transliterated '%s' -> '%s'", englishName, hindiName)

	// Test caching - second call should be instant and return same result
	cachedHindi, err := geminiSvc.TransliterateEnglishToHindi(ctx, englishName)
	if err != nil || cachedHindi != hindiName {
		t.Fatalf("Expected cached result '%s', got '%s' (err: %v)", hindiName, cachedHindi, err)
	}
}

func TestGeminiTransliterationSkipNonEnglish(t *testing.T) {
	geminiSvc := service.NewGeminiService("fake_key", "gemini-3.5-flash-lite")
	ctx := context.Background()

	// Hindi text input should not trigger API call
	res, err := geminiSvc.TransliterateEnglishToHindi(ctx, "रमेश")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if res != "" {
		t.Fatalf("Expected empty response for non-English input, got '%s'", res)
	}
}
