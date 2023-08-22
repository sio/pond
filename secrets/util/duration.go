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
	days, err := strconv.Atoi(dayTags[0][:len(dayTags[0])-1])
	if err != nil {
		return 0, fmt.Errorf("failed to parse day units: %w", err)
	}
	remainder := daysRegex.ReplaceAllString(s, "")
	if len(remainder) == 0 {
		return time.Duration(days) * time.Hour * 24, nil
	}
	result, err = time.ParseDuration(remainder)
	if err != nil {
		return 0, err
	}
	return result + time.Duration(days)*time.Hour*24, nil
}
