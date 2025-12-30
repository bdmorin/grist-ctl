// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

// Common tools
package common

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/Xuanwo/go-locale"
	"github.com/mattn/go-colorable"
	"github.com/muesli/termenv"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/term"
	"golang.org/x/text/language"
)

//go:embed translations/*.json
var translations embed.FS

var localizer *i18n.Localizer // Global localizer
var bundle *i18n.Bundle       // Global bundle

func init() {
	// Detect the language
	tag, err := locale.Detect()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize i18n with English (default) and French languages
	bundle = i18n.NewBundle(language.English)            // Default language
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal) // Register JSON unmarshal function
	if _, err := bundle.LoadMessageFileFS(translations, "translations/en.json"); err != nil {
		log.Printf("Warning: failed to load English translations: %v", err)
	}
	if _, err := bundle.LoadMessageFileFS(translations, "translations/fr.json"); err != nil {
		log.Printf("Warning: failed to load French translations: %v", err)
	}

	localizer = i18n.NewLocalizer(bundle, language.Tag.String(tag)) // Initialize localizer with detected language
}

// Translate a message
func T(msg string) string {
	return localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: msg})
}

// Format string as a title
func Title(txt string) string {
	len := utf8.RuneCountInString(txt)
	line := strings.Repeat("═", len+2)
	newText := fmt.Sprintf("╔%s╗\n║ %s ║\n╚%s╝", line, txt, line)

	return newText
}

// Displays a title
func DisplayTitle(txt string) {
	fmt.Println(Title(txt))
}

// Check if an email is valid
func IsValidEmail(mail string) bool {
	return strings.Contains(mail, "@")
}

// Confirm a question
func Confirm(question string) bool {
	var response string

	fmt.Printf("%s [%s/%s] ", question, T("questions.y"), T("questions.n"))
	_, _ = fmt.Scanln(&response) // Ignore error - empty input is acceptable

	return strings.ToLower(response) == T("questions.y")
}

// Ask a question and return the response
func Ask(question string) string {
	var response string

	fmt.Printf("%s : ", question)
	_, _ = fmt.Scanln(&response) // Ignore error - empty input is acceptable

	return response
}

// AskSecure asks a question and reads the response without echoing to terminal (for passwords/tokens)
func AskSecure(question string) string {
	fmt.Printf("%s : ", question)

	// Read password without echo
	bytePassword, err := term.ReadPassword(syscall.Stdin)
	fmt.Println() // Print newline after password input

	if err != nil {
		log.Printf("Error reading secure input: %v", err)
		return ""
	}

	return string(bytePassword)
}

// NormalizeURL takes any user input URL and normalizes it to https://host.domain.tld format
// Accepts: host.domain.tld, http://host, https://host/, https://host.domain.tld/path, etc.
// Returns: https://host.domain.tld (no trailing slash, no path)
func NormalizeURL(input string) (string, error) {
	// Remove leading/trailing whitespace
	input = strings.TrimSpace(input)

	// If no protocol, add https://
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		input = "https://" + input
	}

	// Parse the URL
	parsedURL, err := url.Parse(input)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %v", err)
	}

	// Validate hostname exists and looks reasonable
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return "", fmt.Errorf("no hostname found in URL")
	}

	// Basic hostname validation (contains at least one dot or is localhost)
	hostnameRegex := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$|^localhost$`)
	if !hostnameRegex.MatchString(hostname) {
		return "", fmt.Errorf("invalid hostname: %s", hostname)
	}

	// Use https by default (even if they provided http)
	scheme := "https"
	if parsedURL.Scheme == "http" {
		scheme = "http"
	}

	// Return normalized URL: scheme://hostname (no trailing slash, no path, no query)
	return fmt.Sprintf("%s://%s", scheme, hostname), nil
}

// Print an example command line
func PrintCommand(txt string) {
	stdout := colorable.NewColorableStdout()

	profile := termenv.ColorProfile()

	if profile != termenv.Ascii {
		cmdText := termenv.String(txt).
			Foreground(termenv.ANSIRed).
			Background(termenv.ANSIWhite).
			String()
		fmt.Fprint(stdout, cmdText)
	} else {
		fmt.Print(txt)
	}
}
