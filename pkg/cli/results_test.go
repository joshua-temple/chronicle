package cli

import (
	"testing"
	"time"
)

func TestParseSince(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkBefore time.Duration // result should be within this duration of now
	}{
		{
			name:        "hours duration",
			input:       "24h",
			expectError: false,
			checkBefore: 25 * time.Hour,
		},
		{
			name:        "minutes duration",
			input:       "30m",
			expectError: false,
			checkBefore: 31 * time.Minute,
		},
		{
			name:        "days duration",
			input:       "7d",
			expectError: false,
			checkBefore: 8 * 24 * time.Hour,
		},
		{
			name:        "date format YYYY-MM-DD",
			input:       "2024-01-01",
			expectError: false,
		},
		{
			name:        "date format with time",
			input:       "2024-01-01T15:04:05",
			expectError: false,
		},
		{
			name:        "RFC3339 format",
			input:       "2024-01-01T15:04:05Z",
			expectError: false,
		},
		{
			name:        "invalid format",
			input:       "invalid-date",
			expectError: true,
		},
		{
			name:        "invalid days format",
			input:       "abcd",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSince(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseSince(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("parseSince(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result.IsZero() {
				t.Errorf("parseSince(%q) returned zero time", tt.input)
			}

			// For duration-based inputs, verify the result is in expected range
			if tt.checkBefore > 0 {
				since := time.Since(result)
				if since > tt.checkBefore {
					t.Errorf("parseSince(%q) returned time %v ago, expected within %v", tt.input, since, tt.checkBefore)
				}
			}
		})
	}
}

func TestParseSinceDaysFormat(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedDays int
		expectError  bool
	}{
		{
			name:         "1 day",
			input:        "1d",
			expectedDays: 1,
			expectError:  false,
		},
		{
			name:         "7 days",
			input:        "7d",
			expectedDays: 7,
			expectError:  false,
		},
		{
			name:         "30 days",
			input:        "30d",
			expectedDays: 30,
			expectError:  false,
		},
		{
			name:         "invalid days",
			input:        "xd",
			expectedDays: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSince(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseSince(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("parseSince(%q) unexpected error: %v", tt.input, err)
				return
			}

			// Calculate expected time
			expectedDuration := time.Duration(tt.expectedDays) * 24 * time.Hour
			expectedTime := time.Now().Add(-expectedDuration)

			// Allow 1 second tolerance
			tolerance := time.Second
			if result.Before(expectedTime.Add(-tolerance)) || result.After(expectedTime.Add(tolerance)) {
				t.Errorf("parseSince(%q) returned %v, expected approximately %v",
					tt.input, result, expectedTime)
			}
		})
	}
}

func TestParseSinceDateFormats(t *testing.T) {
	// Test that various date formats are parsed correctly
	dateInputs := map[string]time.Time{
		"2024-06-15":           time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		"2024-06-15T10:30:00":  time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
		"2024-06-15T10:30:00Z": time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
	}

	for input, expected := range dateInputs {
		t.Run(input, func(t *testing.T) {
			result, err := parseSince(input)
			if err != nil {
				t.Errorf("parseSince(%q) unexpected error: %v", input, err)
				return
			}

			// Compare year, month, day, hour, minute, second (ignoring location)
			if result.Year() != expected.Year() ||
				result.Month() != expected.Month() ||
				result.Day() != expected.Day() {
				t.Errorf("parseSince(%q) date mismatch: got %v, expected %v",
					input, result.Format("2006-01-02"), expected.Format("2006-01-02"))
			}
		})
	}
}
