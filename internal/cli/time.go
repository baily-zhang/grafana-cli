package cli

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var relativeTimePattern = regexp.MustCompile(`^(\d+)(s|sec|secs|second|seconds|m|min|mins|minute|minutes|h|hr|hrs|hour|hours|d|day|days|w|week|weeks)$`)

func normalizeTimeRange(now time.Time, start, end string, defaultWindow time.Duration) (string, string) {
	endValue := normalizeTimeValue(now, end)
	startValue := normalizeTimeValue(now, start)

	base := now.UTC()
	if parsedEnd, err := time.Parse(time.RFC3339, endValue); err == nil {
		base = parsedEnd
	}

	if endValue == "" && defaultWindow > 0 {
		endValue = now.UTC().Format(time.RFC3339)
	}
	if startValue == "" && defaultWindow > 0 {
		startValue = base.Add(-defaultWindow).UTC().Format(time.RFC3339)
	}
	return startValue, endValue
}

func normalizeTimeValue(now time.Time, value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.EqualFold(trimmed, "now") {
		return now.UTC().Format(time.RFC3339)
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "now-") {
		if duration, ok := parseRelativeDuration(strings.TrimPrefix(strings.ToLower(trimmed), "now-")); ok {
			return now.Add(-duration).UTC().Format(time.RFC3339)
		}
	}
	if duration, ok := parseRelativeDuration(trimmed); ok {
		return now.Add(-duration).UTC().Format(time.RFC3339)
	}
	return trimmed
}

func parseRelativeDuration(value string) (time.Duration, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.TrimPrefix(normalized, "-")
	normalized = strings.ReplaceAll(normalized, " ", "")
	matches := relativeTimePattern.FindStringSubmatch(normalized)
	if len(matches) != 3 {
		return 0, false
	}
	amount, err := strconv.Atoi(matches[1])
	if err != nil || amount < 0 {
		return 0, false
	}
	multiplier := time.Second
	switch matches[2] {
	case "s", "sec", "secs", "second", "seconds":
		multiplier = time.Second
	case "m", "min", "mins", "minute", "minutes":
		multiplier = time.Minute
	case "h", "hr", "hrs", "hour", "hours":
		multiplier = time.Hour
	case "d", "day", "days":
		multiplier = 24 * time.Hour
	case "w", "week", "weeks":
		multiplier = 7 * 24 * time.Hour
	}
	return time.Duration(amount) * multiplier, true
}
