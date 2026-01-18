package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// Models to try in order of preference
var modelsToTry = []string{
	"gemini-1.5-flash",   // Current standard (fast, cost-effective)
	"gemini-pro",         // Stable fallback
	"gemini-1.5-pro",     // High intelligence fallback
	"gemini-1.0-pro",     // Legacy fallback
}

// getGeminiClient creates a new Gemini client using the stable SDK
func getGeminiClient(ctx context.Context) (*genai.Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return client, nil
}

// ListAvailableModels lists all available models for debugging
// Call this if translation fails to see what models are accessible
func ListAvailableModels() {
	ctx := context.Background()
	client, err := getGeminiClient(ctx)
	if err != nil {
		log.Printf("❌ ListModels: Failed to create client: %v", err)
		return
	}
	defer client.Close()

	log.Println("📋 Listing available Gemini models...")
	
	iter := client.ListModels(ctx)
	count := 0
	for {
		model, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("❌ ListModels: Error iterating: %v", err)
			break
		}
		
		// Log model details
		log.Printf("  📦 Model: %s", model.Name)
		log.Printf("     Display Name: %s", model.DisplayName)
		log.Printf("     Description: %s", model.Description)
		log.Printf("     Supported Methods: %v", model.SupportedGenerationMethods)
		count++
	}
	
	log.Printf("📋 Total models found: %d", count)
}

// generateWithFallback tries multiple models until one succeeds
func generateWithFallback(ctx context.Context, client *genai.Client, prompt string) (string, error) {
	var lastErr error
	
	for _, modelName := range modelsToTry {
		log.Printf("🔄 Trying model: %s", modelName)
		
		model := client.GenerativeModel(modelName)
		resp, err := model.GenerateContent(ctx, genai.Text(prompt))
		
		if err != nil {
			log.Printf("⚠️ Model %s failed: %v", modelName, err)
			lastErr = err
			continue
		}
		
		// Extract text from response
		responseText := extractTextFromResponse(resp)
		if responseText == "" {
			log.Printf("⚠️ Model %s returned empty response", modelName)
			lastErr = fmt.Errorf("empty response from model %s", modelName)
			continue
		}
		
		log.Printf("✅ Successfully used model: %s", modelName)
		return responseText, nil
	}
	
	// All models failed - list available models for debugging
	log.Println("❌ All models failed! Listing available models for debugging...")
	ListAvailableModels()
	
	return "", fmt.Errorf("all models failed, last error: %w", lastErr)
}

// extractTextFromResponse extracts text content from Gemini response
func extractTextFromResponse(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 {
		return ""
	}
	
	var responseText string
	for _, candidate := range resp.Candidates {
		if candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if textPart, ok := part.(genai.Text); ok {
				responseText += string(textPart)
			}
		}
	}
	
	return responseText
}

// TranslateProduct - Translate product name and description from Uzbek to Russian and English
// Returns maps with keys: "uz", "ru", "en"
func TranslateProduct(nameUz, descUz string) (nameMap, descMap map[string]string, err error) {
	ctx := context.Background()
	client, err := getGeminiClient(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()

	// Prepare prompt
	prompt := fmt.Sprintf(`You are a furniture expert. Translate this product Name and Description from Uzbek to Russian and English. Return ONLY a JSON object in this format:
{
  "name": {
    "uz": "%s",
    "ru": "...",
    "en": "..."
  },
  "description": {
    "uz": "%s",
    "ru": "...",
    "en": "..."
  }
}

Product Name (Uzbek): %s
Product Description (Uzbek): %s

Return ONLY the JSON object, no additional text.`, nameUz, descUz, nameUz, descUz)

	// Generate with fallback
	responseText, err := generateWithFallback(ctx, client, prompt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// Parse JSON response
	var result struct {
		Name        map[string]string `json:"name"`
		Description map[string]string `json:"description"`
	}

	// Clean response text (remove markdown code blocks if present)
	responseText = cleanJSONResponse(responseText)

	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		log.Printf("⚠️ Failed to parse Gemini JSON response: %v\nResponse: %s", err, responseText)
		return nil, nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Ensure all required keys exist
	if result.Name == nil {
		result.Name = make(map[string]string)
	}
	if result.Description == nil {
		result.Description = make(map[string]string)
	}

	// Set Uzbek values (preserve original)
	result.Name["uz"] = nameUz
	result.Description["uz"] = descUz

	// Ensure ru and en exist (fallback to uz if missing)
	if result.Name["ru"] == "" {
		result.Name["ru"] = nameUz
	}
	if result.Name["en"] == "" {
		result.Name["en"] = nameUz
	}
	if result.Description["ru"] == "" {
		result.Description["ru"] = descUz
	}
	if result.Description["en"] == "" {
		result.Description["en"] = descUz
	}

	return result.Name, result.Description, nil
}

// TranslateShop - Translate shop name, description, and address from Uzbek to Russian and English
// Returns maps with keys: "uz", "ru", "en"
func TranslateShop(nameUz, descUz, addrUz string) (nameMap, descMap, addrMap map[string]string, err error) {
	ctx := context.Background()
	client, err := getGeminiClient(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	defer client.Close()

	// Prepare prompt
	prompt := fmt.Sprintf(`You are a furniture shop expert. Translate this shop Name, Description, and Address from Uzbek to Russian and English. Return ONLY a JSON object in this format:
{
  "name": {
    "uz": "%s",
    "ru": "...",
    "en": "..."
  },
  "description": {
    "uz": "%s",
    "ru": "...",
    "en": "..."
  },
  "address": {
    "uz": "%s",
    "ru": "...",
    "en": "..."
  }
}

Shop Name (Uzbek): %s
Shop Description (Uzbek): %s
Shop Address (Uzbek): %s

Return ONLY the JSON object, no additional text.`, nameUz, descUz, addrUz, nameUz, descUz, addrUz)

	// Generate with fallback
	responseText, err := generateWithFallback(ctx, client, prompt)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// Parse JSON response
	var result struct {
		Name        map[string]string `json:"name"`
		Description map[string]string `json:"description"`
		Address     map[string]string `json:"address"`
	}

	// Clean response text (remove markdown code blocks if present)
	responseText = cleanJSONResponse(responseText)

	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		log.Printf("⚠️ Failed to parse Gemini JSON response: %v\nResponse: %s", err, responseText)
		return nil, nil, nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Ensure all required keys exist
	if result.Name == nil {
		result.Name = make(map[string]string)
	}
	if result.Description == nil {
		result.Description = make(map[string]string)
	}
	if result.Address == nil {
		result.Address = make(map[string]string)
	}

	// Set Uzbek values (preserve original)
	result.Name["uz"] = nameUz
	if descUz != "" {
		result.Description["uz"] = descUz
	}
	if addrUz != "" {
		result.Address["uz"] = addrUz
	}

	// Ensure ru and en exist (fallback to uz if missing)
	if result.Name["ru"] == "" {
		result.Name["ru"] = nameUz
	}
	if result.Name["en"] == "" {
		result.Name["en"] = nameUz
	}
	if result.Description["ru"] == "" && descUz != "" {
		result.Description["ru"] = descUz
	}
	if result.Description["en"] == "" && descUz != "" {
		result.Description["en"] = descUz
	}
	if result.Address["ru"] == "" && addrUz != "" {
		result.Address["ru"] = addrUz
	}
	if result.Address["en"] == "" && addrUz != "" {
		result.Address["en"] = addrUz
	}

	return result.Name, result.Description, result.Address, nil
}

// cleanJSONResponse - Remove markdown code blocks and extra whitespace from JSON response
func cleanJSONResponse(text string) string {
	// Remove markdown code blocks (```json ... ``` or ``` ... ```)
	text = removeMarkdownCodeBlocks(text)

	// Trim whitespace
	text = trimWhitespace(text)

	return text
}

// removeMarkdownCodeBlocks - Remove markdown code block markers
func removeMarkdownCodeBlocks(text string) string {
	// Remove ```json at the start
	if len(text) >= 7 && text[:7] == "```json" {
		text = text[7:]
	} else if len(text) >= 3 && text[:3] == "```" {
		text = text[3:]
	}

	// Remove ``` at the end
	if len(text) >= 3 && text[len(text)-3:] == "```" {
		text = text[:len(text)-3]
	}

	return text
}

// trimWhitespace - Trim leading and trailing whitespace
func trimWhitespace(text string) string {
	// Remove leading whitespace
	for len(text) > 0 && (text[0] == ' ' || text[0] == '\n' || text[0] == '\r' || text[0] == '\t') {
		text = text[1:]
	}

	// Remove trailing whitespace
	for len(text) > 0 && (text[len(text)-1] == ' ' || text[len(text)-1] == '\n' || text[len(text)-1] == '\r' || text[len(text)-1] == '\t') {
		text = text[:len(text)-1]
	}

	return text
}
