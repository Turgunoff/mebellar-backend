package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// TranslateProduct - Translate product name and description from Uzbek to Russian and English
// Returns maps with keys: "uz", "ru", "en"
func TranslateProduct(nameUz, descUz string) (nameMap, descMap map[string]string, err error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}

	// Initialize Gemini client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Gemini client: %w", err)
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

	// Get the model
	model := client.GenerativeModel("gemini-pro")
	
	// Generate content
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// Extract text from response
	var responseText string
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		if textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
			responseText = string(textPart)
		}
	}

	if responseText == "" {
		return nil, nil, fmt.Errorf("empty response from Gemini")
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
