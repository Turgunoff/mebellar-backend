package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// DeepSeek API configuration
const (
	deepSeekBaseURL = "https://api.deepseek.com"
	deepSeekModel   = "deepseek-chat" // DeepSeek-V3
)

// getDeepSeekClient creates a new DeepSeek client using OpenAI-compatible SDK
func getDeepSeekClient() *openai.Client {
	apiKey := os.Getenv("OPENAI_API_KEY") // DeepSeek key stored here
	if apiKey == "" {
		log.Println("⚠️ OPENAI_API_KEY (DeepSeek) environment variable is not set")
		return nil
	}

	// Configure client to use DeepSeek API endpoint
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = deepSeekBaseURL

	return openai.NewClientWithConfig(config)
}

// generateTranslation calls DeepSeek API to generate translation
func generateTranslation(systemPrompt, userPrompt string) (string, error) {
	client := getDeepSeekClient()
	if client == nil {
		return "", fmt.Errorf("OPENAI_API_KEY (DeepSeek) environment variable is not set")
	}

	ctx := context.Background()

	// Create chat completion request
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: deepSeekModel, // DeepSeek-V3
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
			Temperature: 0.3, // Lower temperature for more consistent translations
			MaxTokens:   1000,
		},
	)

	if err != nil {
		return "", fmt.Errorf("DeepSeek API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from DeepSeek")
	}

	responseText := resp.Choices[0].Message.Content
	log.Printf("✅ DeepSeek response received (model: %s, tokens: %d)", resp.Model, resp.Usage.TotalTokens)

	return responseText, nil
}

// TranslateProduct - Translate product name and description from Uzbek to Russian and English
// Returns maps with keys: "uz", "ru", "en"
func TranslateProduct(nameUz, descUz string) (nameMap, descMap map[string]string, err error) {
	systemPrompt := `You are a professional translator specializing in furniture and home goods. 
Output ONLY valid JSON without any markdown formatting or code blocks.
Translate accurately while preserving the original meaning and tone.`

	userPrompt := fmt.Sprintf(`Translate this product Name and Description from Uzbek to Russian and English.
Return ONLY a JSON object in this EXACT format (no markdown, no code blocks):
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
Product Description (Uzbek): %s`, nameUz, descUz, nameUz, descUz)

	responseText, err := generateTranslation(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("⚠️ TranslateProduct failed: %v", err)
		// Fallback: return original Uzbek text for all languages
		return createFallbackName(nameUz), createFallbackDesc(descUz), nil
	}

	// Parse JSON response
	var result struct {
		Name        map[string]string `json:"name"`
		Description map[string]string `json:"description"`
	}

	// Clean response text (remove markdown code blocks if present)
	responseText = cleanJSONResponse(responseText)

	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		log.Printf("⚠️ Failed to parse DeepSeek JSON response: %v\nResponse: %s", err, responseText)
		// Fallback: return original Uzbek text for all languages
		return createFallbackName(nameUz), createFallbackDesc(descUz), nil
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

	log.Printf("✅ Product translated: %s -> ru:%s, en:%s", nameUz, result.Name["ru"], result.Name["en"])

	return result.Name, result.Description, nil
}

// TranslateShop - Translate shop name, description, and address from Uzbek to Russian and English
// Returns maps with keys: "uz", "ru", "en"
func TranslateShop(nameUz, descUz, addrUz string) (nameMap, descMap, addrMap map[string]string, err error) {
	systemPrompt := `You are a professional translator specializing in business and location names.
Output ONLY valid JSON without any markdown formatting or code blocks.
Translate accurately while preserving the original meaning. For addresses, transliterate proper nouns.`

	userPrompt := fmt.Sprintf(`Translate this shop Name, Description, and Address from Uzbek to Russian and English.
Return ONLY a JSON object in this EXACT format (no markdown, no code blocks):
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
Shop Address (Uzbek): %s`, nameUz, descUz, addrUz, nameUz, descUz, addrUz)

	responseText, err := generateTranslation(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("⚠️ TranslateShop failed: %v", err)
		// Fallback: return original Uzbek text for all languages
		return createFallbackName(nameUz), createFallbackDesc(descUz), createFallbackAddr(addrUz), nil
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
		log.Printf("⚠️ Failed to parse DeepSeek JSON response: %v\nResponse: %s", err, responseText)
		// Fallback: return original Uzbek text for all languages
		return createFallbackName(nameUz), createFallbackDesc(descUz), createFallbackAddr(addrUz), nil
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

	log.Printf("✅ Shop translated: %s -> ru:%s, en:%s", nameUz, result.Name["ru"], result.Name["en"])

	return result.Name, result.Description, result.Address, nil
}

// Fallback helper functions
func createFallbackName(nameUz string) map[string]string {
	return map[string]string{
		"uz": nameUz,
		"ru": nameUz,
		"en": nameUz,
	}
}

func createFallbackDesc(descUz string) map[string]string {
	if descUz == "" {
		return map[string]string{}
	}
	return map[string]string{
		"uz": descUz,
		"ru": descUz,
		"en": descUz,
	}
}

func createFallbackAddr(addrUz string) map[string]string {
	if addrUz == "" {
		return map[string]string{}
	}
	return map[string]string{
		"uz": addrUz,
		"ru": addrUz,
		"en": addrUz,
	}
}

// cleanJSONResponse - Remove markdown code blocks and extra whitespace from JSON response
func cleanJSONResponse(text string) string {
	text = strings.TrimSpace(text)

	// Remove ```json at the start
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
	}

	// Remove ``` at the end
	if strings.HasSuffix(text, "```") {
		text = strings.TrimSuffix(text, "```")
	}

	return strings.TrimSpace(text)
}
