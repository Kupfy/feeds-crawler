package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var unicodeFractions = map[rune]float64{
	'¼': 0.25,
	'½': 0.5,
	'¾': 0.75,
	'⅐': 1.0 / 7.0,
	'⅑': 1.0 / 9.0,
	'⅒': 0.1,
	'⅓': 1.0 / 3.0,
	'⅔': 2.0 / 3.0,
	'⅕': 0.2,
	'⅖': 0.4,
	'⅗': 0.6,
	'⅘': 0.8,
	'⅙': 1.0 / 6.0,
	'⅚': 5.0 / 6.0,
	'⅛': 0.125,
	'⅜': 0.375,
	'⅝': 0.625,
	'⅞': 0.875,
}

var fractionUnicodes = map[float64]rune{
	0.25:      '¼',
	0.5:       '½',
	0.75:      '¾',
	1.0 / 7.0: '⅐',
	1.0 / 9.0: '⅑',
	0.1:       '⅒',
	1.0 / 3.0: '⅓',
	2.0 / 3.0: '⅔',
	0.2:       '⅕',
	0.4:       '⅖',
	0.6:       '⅗',
	0.8:       '⅘',
	1.0 / 6.0: '⅙',
	5.0 / 6.0: '⅚',
	0.125:     '⅛',
	0.375:     '⅜',
	0.625:     '⅝',
	0.875:     '⅞',
}

var (
	// Matches "1 1/2" (mixed number)
	mixedNumberPattern = regexp.MustCompile(`^(\d+)\s+(\d+)/(\d+)$`)

	// Matches "3/4" (simple fraction)
	simpleFractionPattern = regexp.MustCompile(`^(\d+)/(\d+)$`)

	// Matches "1.5" or "1" (decimal or whole number)
	decimalPattern = regexp.MustCompile(`^\d+\.?\d*$`)
)

func Tokenize(s string) []string {
	s = strings.ReplaceAll(s, ",", " , ")
	s = strings.ReplaceAll(s, "(", " ( ")
	s = strings.ReplaceAll(s, ")", " ) ")
	return strings.Fields(s)
}

func ParseQuantity(s string) (float64, error) {
	s = strings.TrimSpace(s)

	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	// Check for Unicode fraction characters first
	if val, ok := parseUnicodeFraction(s); ok {
		return val, nil
	}

	// Check for mixed numbers: "1 1/2"
	if matches := mixedNumberPattern.FindStringSubmatch(s); matches != nil {
		whole, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid whole number: %w", err)
		}

		numerator, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid numerator: %w", err)
		}

		denominator, err := strconv.ParseFloat(matches[3], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid denominator: %w", err)
		}

		if denominator == 0 {
			return 0, fmt.Errorf("division by zero")
		}

		return whole + (numerator / denominator), nil
	}

	// Check for simple fractions: "3/4"
	if matches := simpleFractionPattern.FindStringSubmatch(s); matches != nil {
		numerator, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid numerator: %w", err)
		}

		denominator, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid denominator: %w", err)
		}

		if denominator == 0 {
			return 0, fmt.Errorf("division by zero")
		}

		return numerator / denominator, nil
	}

	// Check for decimal or whole number: "1.5" or "1"
	if decimalPattern.MatchString(s) {
		return strconv.ParseFloat(s, 64)
	}

	// Try to handle mixed Unicode fractions like "1½" or "2¾"
	if val, ok := parseMixedUnicodeFraction(s); ok {
		return val, nil
	}

	return 0, fmt.Errorf("unable to parse quantity: %s", s)
}

// parseUnicodeFraction checks if the entire string is a single Unicode fraction
func parseUnicodeFraction(s string) (float64, bool) {
	if len(s) == 0 {
		return 0, false
	}

	// Check if it's a single Unicode fraction character
	runes := []rune(s)
	if len(runes) == 1 {
		if val, ok := unicodeFractions[runes[0]]; ok {
			return val, true
		}
	}

	return 0, false
}

// parseMixedUnicodeFraction handles "1½", "2¾", etc.
func parseMixedUnicodeFraction(s string) (float64, bool) {
	runes := []rune(s)
	if len(runes) < 2 {
		return 0, false
	}

	// Find where the Unicode fraction starts
	var wholeStr string
	var fractionRune rune
	foundFraction := false

	for i, r := range runes {
		if _, isFraction := unicodeFractions[r]; isFraction {
			wholeStr = string(runes[:i])
			fractionRune = r
			foundFraction = true
			break
		}
	}

	if !foundFraction {
		return 0, false
	}

	// Parse the whole number part
	wholeStr = strings.TrimSpace(wholeStr)
	if wholeStr == "" {
		// Just a fraction like "½"
		return unicodeFractions[fractionRune], true
	}

	whole, err := strconv.ParseFloat(wholeStr, 64)
	if err != nil {
		return 0, false
	}

	fraction := unicodeFractions[fractionRune]
	return whole + fraction, true
}

func FractionPrettyPrint(num *float64) string {
	if num == nil {
		return ""
	}
	wholeNum := int(*num)
	fraction := *num - float64(wholeNum)
	return fmt.Sprintf("%d%s", wholeNum, fractionUnicodes[fraction])
}
