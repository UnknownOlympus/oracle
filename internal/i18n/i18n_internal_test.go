package i18n

import (
	"testing"
)

func TestNewLocalizer(t *testing.T) {
	localizer, err := NewLocalizer()
	if err != nil {
		t.Fatalf("Failed to create localizer: %v", err)
	}

	if localizer == nil {
		t.Fatal("Localizer is nil")
	}

	if len(localizer.translations) == 0 {
		t.Fatal("No translations loaded")
	}

	// Check that both languages are loaded
	if _, ok := localizer.translations["en"]; !ok {
		t.Error("English translations not loaded")
	}

	if _, ok := localizer.translations["uk"]; !ok {
		t.Error("Ukrainian translations not loaded")
	}
}

func TestGet(t *testing.T) {
	localizer, err := NewLocalizer()
	if err != nil {
		t.Fatalf("Failed to create localizer: %v", err)
	}

	tests := []struct {
		name     string
		lang     string
		key      string
		expected string
	}{
		{
			name:     "English welcome message",
			lang:     "en",
			key:      "welcome.authenticated",
			expected: "🤡 Welcome to the almshouse, slave of Radionet!",
		},
		{
			name:     "Ukrainian welcome message",
			lang:     "uk",
			key:      "welcome.authenticated",
			expected: "🤡 Ласкаво просимо до богодєльні, раб Радіонету!",
		},
		{
			name:     "Fallback to English",
			lang:     "unknown",
			key:      "welcome.authenticated",
			expected: "🤡 Welcome to the almshouse, slave of Radionet!",
		},
		{
			name:     "Non-existent key returns key itself",
			lang:     "en",
			key:      "non.existent.key",
			expected: "non.existent.key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := localizer.Get(tt.lang, tt.key)
			if result != tt.expected {
				t.Errorf("Get(%q, %q) = %q, want %q", tt.lang, tt.key, result, tt.expected)
			}
		})
	}
}

func TestGetWithData(t *testing.T) {
	localizer, err := NewLocalizer()
	if err != nil {
		t.Fatalf("Failed to create localizer: %v", err)
	}

	tests := []struct {
		name     string
		lang     string
		key      string
		data     map[string]interface{}
		expected string
	}{
		{
			name: "Replace single placeholder in English",
			lang: "en",
			key:  "info.name",
			data: map[string]interface{}{
				"name": "John Doe",
			},
			expected: "*Name:* John Doe",
		},
		{
			name: "Replace single placeholder in Ukrainian",
			lang: "uk",
			key:  "info.name",
			data: map[string]interface{}{
				"name": "Іван Петренко",
			},
			expected: "*Ім'я:* Іван Петренко",
		},
		{
			name: "Replace multiple placeholders",
			lang: "en",
			key:  "statistic.total",
			data: map[string]interface{}{
				"type":  "Total",
				"count": 42,
			},
			expected: "👑 Total: 42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := localizer.GetWithData(tt.lang, tt.key, tt.data)
			if result != tt.expected {
				t.Errorf("GetWithData(%q, %q, %v) = %q, want %q", tt.lang, tt.key, tt.data, result, tt.expected)
			}
		})
	}
}

func TestNormalizeLanguageCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "English",
			input:    "en",
			expected: "en",
		},
		{
			name:     "English with region",
			input:    "en-US",
			expected: "en",
		},
		{
			name:     "Ukrainian (uk)",
			input:    "uk",
			expected: "uk",
		},
		{
			name:     "Ukrainian (ua)",
			input:    "ua",
			expected: "uk",
		},
		{
			name:     "Unknown language defaults to English",
			input:    "de",
			expected: "en",
		},
		{
			name:     "Empty string defaults to English",
			input:    "",
			expected: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeLanguageCode(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeLanguageCode(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
