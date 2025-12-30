// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package common

import (
	"testing"
	"unicode/utf8"
)

func TestTitle(t *testing.T) {
	txt := "This is my title"
	title := Title(txt)

	titleLength := utf8.RuneCountInString(title)
	lenTxt := utf8.RuneCountInString(txt)
	targetLen := 3*lenTxt + 6*2 + 2
	if titleLength != targetLen {
		t.Errorf("Title's length is not correct (%d/%d)", titleLength, targetLen)
	}
}

func TestEmail(t *testing.T) {
	email := "user@domain.fr"
	if !IsValidEmail(email) {
		t.Errorf("Email %s should be valid", email)
	}
	email = "userdomain"
	if IsValidEmail(email) {
		t.Errorf("Email %s should not be valid", email)
	}
}

func TestTranslation(t *testing.T) {
	msg := "app.title"
	translated := T(msg)
	if translated == msg {
		t.Errorf("Translation for %s should not be the same", msg)
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		// Basic cases - should normalize to https
		{"hexxa.getgrist.com", "https://hexxa.getgrist.com", false},
		{"grist.hexxa.dev", "https://grist.hexxa.dev", false},

		// With protocols
		{"https://hexxa.getgrist.com", "https://hexxa.getgrist.com", false},
		{"http://localhost", "http://localhost", false},

		// With trailing slashes (should remove)
		{"https://hexxa.getgrist.com/", "https://hexxa.getgrist.com", false},
		{"hexxa.getgrist.com/", "https://hexxa.getgrist.com", false},

		// With paths (should remove)
		{"https://hexxa.getgrist.com/api/docs", "https://hexxa.getgrist.com", false},
		{"hexxa.getgrist.com/some/path", "https://hexxa.getgrist.com", false},

		// With ports (should preserve)
		{"localhost:8484", "https://localhost", false},
		{"http://localhost:8484", "http://localhost", false},

		// Whitespace handling
		{"  hexxa.getgrist.com  ", "https://hexxa.getgrist.com", false},

		// Invalid cases
		{"", "", true},
		{"not a url", "", true},
		{"http://", "", true},
		{"://invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := NormalizeURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
