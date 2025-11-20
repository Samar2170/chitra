package main

import (
	"log"
	"testing"
	"time"
)

func TestParseDeadline(t *testing.T) {
	var testCases = []struct {
		input    string
		expected time.Time
		errorStr string
	}{
		{"week", time.Now().Local().Add(time.Hour * 24 * 7), ""},
		{"day", time.Now().Local().Add(time.Hour * 24), ""},
		{"month", time.Now().Local().Add(time.Hour * 24 * 30), ""},
		{"quarter", time.Now().Local().Add(time.Hour * 24 * 90), ""},

		{"25feb", time.Date(2026, 2, 25, 0, 0, 0, 0, time.Local), ""},
		{"32feb", time.Time{}, "invalid day"},
		{"1mar", time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local), ""},
		{"30dec", time.Date(2025, 12, 30, 0, 0, 0, 0, time.Local), ""},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual, err := ParseDeadline(tc.input)
			log.Println(actual, err)
			if err != nil {
				errorStr := err.Error()
				if errorStr != tc.errorStr {
					log.Println(errorStr)
					t.Errorf("expected %v, got %v", tc.errorStr, errorStr)
				} else {
					return
				}
			}
			// validate date only
			tc.expected = time.Date(tc.expected.Year(), tc.expected.Month(), tc.expected.Day(), 0, 0, 0, 0, time.Local)
			if actual != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, actual)
			}
		})
	}
}
