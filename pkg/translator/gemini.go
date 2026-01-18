package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"google.golang.org/genai"
)

// getGeminiClient creates a new Gemini client using the new SDK
func getGeminiClient(ctx context.Context) (*genai.Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return client, nil
}

// TranslateProduct - Translate product name and description from Uzbek to Russian and English
// Returns maps with keys: "uz", "ru", "en"
func TranslateProduct(nameUz, descUz string) (nameMap, descMap map[string]string, err error) {
	ctx := context.Background()
	client, err := getGeminiClient(ctx)
	if err != nil {
		return nil, nil, err
	}

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

	// Use new SDK API - try multiple model names as fallback
	// The new SDK might use different model names than expected
	modelsToTry := []string{"gemini-1.0-pro", "gemini-pro", "gemini-1.5-pro"}
	var result *genai.GenerateContentResult
	var lastErr error
	
	for _, modelName := range modelsToTry {
		result, lastErr = client.Models.GenerateContent(
			ctx,
			modelName,
			genai.Text(prompt),
			nil,
		)
		if lastErr == nil {
			log.Printf("✅ Successfully used model: %s", modelName)
			break
		}
		log.Printf("⚠️ Model %s failed: %v, trying next...", modelName, lastErr)
	}
	
	if lastErr != nil {
		return nil, nil, fmt.Errorf("failed to generate content with any model: %w", lastErr)
	}

	responseText := result.Text()
	if responseText == "" {
		return nil, nil, fmt.Errorf("empty response from Gemini")
	}

	// Parse JSON response
	var parsed struct {
		Name        map[string]string `json:"name"`
		Description map[string]string `json:"description"`
	}

	// Clean response text (remove markdown code blocks if present)
	responseText = cleanJSONResponse(responseText)

	if err := json.Unmarshal([]byte(responseText), &parsed); err != nil {
		log.Printf("⚠️ Failed to parse Gemini JSON response: %v\nResponse: %s", err, responseText)
		return nil, nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Ensure all required keys exist
	if parsed.Name == nil {
		parsed.Name = make(map[string]string)
	}
	if parsed.Description == nil {
		parsed.Description = make(map[string]string)
	}

	// Set Uzbek values (preserve original)
	parsed.Name["uz"] = nameUz
	parsed.Description["uz"] = descUz

	// Ensure ru and en exist (fallback to uz if missing)
	if parsed.Name["ru"] == "" {
		parsed.Name["ru"] = nameUz
	}
	if parsed.Name["en"] == "" {
		parsed.Name["en"] = nameUz
	}
	if parsed.Description["ru"] == "" {
		parsed.Description["ru"] = descUz
	}
	if parsed.Description["en"] == "" {
		parsed.Description["en"] = descUz
	}

	return parsed.Name, parsed.Description, nil
}

// TranslateShop - Translate shop name, description, and address from Uzbek to Russian and English
// Returns maps with keys: "uz", "ru", "en"
func TranslateShop(nameUz, descUz, addrUz string) (nameMap, descMap, addrMap map[string]string, err error) {
	ctx := context.Background()
	client, err := getGeminiClient(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

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

	// Use new SDK API - try multiple model names as fallback
	// The new SDK might use different model names than expected
	modelsToTry := []string{"gemini-1.0-pro", "gemini-pro", "gemini-1.5-pro"}
	var result *genai.GenerateContentResult
	var lastErr error
	
	for _, modelName := range modelsToTry {
		result, lastErr = client.Models.GenerateContent(
			ctx,
			modelName,
			genai.Text(prompt),
			nil,
		)
		if lastErr == nil {
			log.Printf("✅ Successfully used model: %s", modelName)
			break
		}
		log.Printf("⚠️ Model %s failed: %v, trying next...", modelName, lastErr)
	}
	
	if lastErr != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate content with any model: %w", lastErr)
	}

	responseText := result.Text()
	if responseText == "" {
		return nil, nil, nil, fmt.Errorf("empty response from Gemini")
	}

	// Parse JSON response
	var parsed struct {
		Name        map[string]string `json:"name"`
		Description map[string]string `json:"description"`
		Address     map[string]string `json:"address"`
	}

	// Clean response text (remove markdown code blocks if present)
	responseText = cleanJSONResponse(responseText)

	if err := json.Unmarshal([]byte(responseText), &parsed); err != nil {
		log.Printf("⚠️ Failed to parse Gemini JSON response: %v\nResponse: %s", err, responseText)
		return nil, nil, nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Ensure all required keys exist
	if parsed.Name == nil {
		parsed.Name = make(map[string]string)
	}
	if parsed.Description == nil {
		parsed.Description = make(map[string]string)
	}
	if parsed.Address == nil {
		parsed.Address = make(map[string]string)
	}

	// Set Uzbek values (preserve original)
	parsed.Name["uz"] = nameUz
	if descUz != "" {
		parsed.Description["uz"] = descUz
	}
	if addrUz != "" {
		parsed.Address["uz"] = addrUz
	}

	// Ensure ru and en exist (fallback to uz if missing)
	if parsed.Name["ru"] == "" {
		parsed.Name["ru"] = nameUz
	}
	if parsed.Name["en"] == "" {
		parsed.Name["en"] = nameUz
	}
	if parsed.Description["ru"] == "" && descUz != "" {
		parsed.Description["ru"] = descUz
	}
	if parsed.Description["en"] == "" && descUz != "" {
		parsed.Description["en"] = descUz
	}
	if parsed.Address["ru"] == "" && addrUz != "" {
		parsed.Address["ru"] = addrUz
	}
	if parsed.Address["en"] == "" && addrUz != "" {
		parsed.Address["en"] = addrUz
	}

	return parsed.Name, parsed.Description, parsed.Address, nil
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
