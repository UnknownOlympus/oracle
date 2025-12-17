package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed locales/*.json
var localesFS embed.FS

// Localizer handles translation for different languages.
type Localizer struct {
	translations map[string]map[string]string
	mu           sync.RWMutex
}

// NewLocalizer creates a new Localizer instance and loads all translations.
func NewLocalizer() (*Localizer, error) {
	locale := &Localizer{
		translations: make(map[string]map[string]string),
	}

	// Load supported languages
	languages := []string{"en", "uk"}
	for _, lang := range languages {
		if err := locale.loadLanguage(lang); err != nil {
			return nil, fmt.Errorf("failed to load language %s: %w", lang, err)
		}
	}

	return locale, nil
}

// loadLanguage loads translations for a specific language from embedded JSON files.
func (l *Localizer) loadLanguage(lang string) error {
	filename := fmt.Sprintf("locales/%s.json", lang)
	data, err := localesFS.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read locale file %s: %w", filename, err)
	}

	var translations map[string]string
	if err = json.Unmarshal(data, &translations); err != nil {
		return fmt.Errorf("failed to unmarshal locale file %s: %w", filename, err)
	}

	l.mu.Lock()
	l.translations[lang] = translations
	l.mu.Unlock()

	return nil
}

// Get returns the translation for the given key in the specified language.
// If the translation is not found, it returns the key itself.
func (l *Localizer) Get(lang, key string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if langTranslations, ok := l.translations[lang]; ok {
		if translation, exists := langTranslations[key]; exists {
			return translation
		}
	}

	// Fallback to English if translation not found
	if lang != "en" {
		if enTranslations, ok := l.translations["en"]; ok {
			if translation, exists := enTranslations[key]; exists {
				return translation
			}
		}
	}

	// Return the key itself if no translation found
	return key
}

// GetWithData returns the translation for the given key with placeholder replacement.
// Example: GetWithData("en", "welcome.user", map[string]string{"name": "John"}).
func (l *Localizer) GetWithData(lang, key string, data map[string]interface{}) string {
	translation := l.Get(lang, key)

	// Simple placeholder replacement
	for k, v := range data {
		placeholder := fmt.Sprintf("{%s}", k)
		translation = replaceAll(translation, placeholder, fmt.Sprintf("%v", v))
	}

	return translation
}

// replaceAll is a helper function to replace all occurrences of old with new in s.
func replaceAll(str, oldValue, newValue string) string {
	result := ""
	for {
		idx := indexOf(str, oldValue)
		if idx == -1 {
			result += str
			break
		}
		result += str[:idx] + newValue
		str = str[idx+len(oldValue):]
	}
	return result
}

// indexOf returns the index of the first occurrence of substr in s, or -1 if not present.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// NormalizeLanguageCode normalizes Telegram language codes to our supported languages.
func NormalizeLanguageCode(telegramLang string) string {
	if telegramLang == "" {
		return "en"
	}

	// Handle language codes like "en-US" -> "en"
	const langCodeShortLength = 2
	if len(telegramLang) >= langCodeShortLength {
		langCode := telegramLang[:2]

		// Map to supported languages
		switch langCode {
		case "en":
			return "en"
		case "uk", "ua": // Both uk and ua map to Ukrainian
			return "uk"
		default:
			return "en" // Default to English
		}
	}

	return "en"
}
