package util

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseDuration parses human-ish time strings into time.Duration.
func ParseDuration(input string) (time.Duration, error) {
	input = strings.ToLower(input)

	re := regexp.MustCompile(`([\d.]+)\s*(h|hr|hrs|hour|hours|m|min|mins|minute|minutes|s|sec|secs|second|seconds)`)
	matches := re.FindAllStringSubmatch(input, -1)

	if len(matches) == 0 {
		return 0, errors.New("no time components found")
	}

	var duration time.Duration

	for _, m := range matches {
		value, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, err
		}

		switch m[2] {
		case "h", "hr", "hrs", "hour", "hours":
			duration += time.Duration(value * float64(time.Hour))
		case "m", "min", "mins", "minute", "minutes":
			duration += time.Duration(value * float64(time.Minute))
		case "s", "sec", "secs", "second", "seconds":
			duration += time.Duration(value * float64(time.Second))
		}
	}

	return duration, nil
}

func ExtractNumber(text string) (float64, error) {
	re := regexp.MustCompile(`[-+]?\d*\.?\d+`)
	match := re.FindString(text)

	if match == "" {
		return 0, errors.New("no number found in text")
	}

	return strconv.ParseFloat(match, 64)
}
