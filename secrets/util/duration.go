package util

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var daysRegex = regexp.MustCompile(`[0-9]+d`)

// Add days ("d") as unit to time.Duration
func ParseDuration(s string) (time.Duration, error) {
	result, err := time.ParseDuration(s)
	if err == nil {
		return result, nil
	}
	dayTags := daysRegex.FindAllString(s, 2)
	if len(dayTags) == 0 {
		return 0, err
	}
	if len(dayTags) > 1 {
		return 0, fmt.Errorf("day unit appears multiple times in duration string: %s", s)
	}
	daysCount, err := strconv.Atoi(dayTags[0][:len(dayTags[0])-1])
	if err != nil {
		return 0, fmt.Errorf("failed to parse day units: %w", err)
	}
	days := time.Duration(daysCount) * time.Hour * 24
	remainder := daysRegex.ReplaceAllString(s, "")
	var sign time.Duration = 1
	if len(remainder) > 0 && remainder[0] == '-' {
		sign = -1
		remainder = remainder[1:]
	}
	if len(remainder) == 0 {
		return sign * days, nil
	}
	result, err = time.ParseDuration(remainder)
	if err != nil {
		return 0, err
	}
	return sign * (result + days), nil
}
