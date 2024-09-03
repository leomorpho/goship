package emojirepo_test

import (
	"testing"

	emojirepo "github.com/mikestefanello/pagoda/pkg/repos/emoji"
)

func TestGetRootEmojiFromShortcode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "root emoji without modifier",
			input:    ":smile:",
			expected: ":smile:",
		},
		{
			name:     "emoji with skin tone modifier",
			input:    ":angel::skin-tone-4:",
			expected: ":angel:",
		},
		{
			name:     "emoji with unsupported modifier",
			input:    ":woman-gesturing-ok::skin-tone-2:",
			expected: ":woman-gesturing-ok:",
		},
		{
			name:     "emoji with unsupported modifier",
			input:    ":+1::skin-tone-4:",
			expected: ":+1:",
		},

		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := emojirepo.GetRootEmojiFromShortcode(tt.input)
			if result != tt.expected {
				t.Errorf("GetRootEmojiFromShortcode(%s) got %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetRootEmojiFromUnifiedCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple emoji code",
			input:    "1f600",
			expected: "1f600",
		},
		{
			name:     "emoji code with skin tone modifier",
			input:    "1f471-1f3fb",
			expected: "1f471",
		},
		{
			name:     "emoji code with multiple modifiers",
			input:    "1f937-1f3fc-200d-2642-fe0f",
			expected: "1f937",
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := emojirepo.GetRootEmojiFromUnifiedCode(tt.input)
			if result != tt.expected {
				t.Errorf("GetRootEmojiFromUnifiedCode(%s) got %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}
