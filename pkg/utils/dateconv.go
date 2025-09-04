package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ConvertProjectsDateToISO converts GitHub Projects date format to ISO date format
// Examples:
//
//	@today -> 2025-09-04
//	@today-1d -> 2025-09-03
//	>@today-1w -> >2025-08-28
//	-@today -> -2025-09-04
func ConvertProjectsDateToISO(input string) (string, error) {
	return ConvertProjectsDateToISOWithBase(input, time.Now())
}

// ConvertProjectsDateToISOWithBase converts with a specific base date (for testing)
func ConvertProjectsDateToISOWithBase(input string, baseDate time.Time) (string, error) {
	// Remove spaces for easier parsing
	input = strings.ReplaceAll(input, " ", "")

	// Handle exclusion prefix
	exclude := false
	if strings.HasPrefix(input, "-") {
		exclude = true
		input = input[1:]
	}

	// Handle comparison operators
	operator := ""
	if strings.HasPrefix(input, ">") {
		operator = ">"
		input = input[1:]
		if strings.HasPrefix(input, "=") {
			operator = ">="
			input = input[1:]
		}
	} else if strings.HasPrefix(input, "<") {
		operator = "<"
		input = input[1:]
		if strings.HasPrefix(input, "=") {
			operator = "<="
			input = input[1:]
		}
	}

	// Parse the date expression
	isoDate, err := parseDate(input, baseDate)
	if err != nil {
		return "", err
	}

	// Construct the result
	result := ""
	if exclude {
		result += "-"
	}
	result += operator + isoDate

	return result, nil
}

// parseDate parses various date formats
func parseDate(input string, baseDate time.Time) (string, error) {
	// Handle @today format
	if input == "@today" {
		return baseDate.Format("2006-01-02"), nil
	}

	// Handle @today+Nd or @today-Nd format
	todayPattern := regexp.MustCompile(`^@today([+-])(\d+)([dw])$`)
	if matches := todayPattern.FindStringSubmatch(input); matches != nil {
		sign := matches[1]
		numStr := matches[2]
		unit := matches[3]

		num, err := strconv.Atoi(numStr)
		if err != nil {
			return "", fmt.Errorf("invalid number: %s", numStr)
		}

		var duration time.Duration
		switch unit {
		case "d":
			duration = time.Duration(num) * 24 * time.Hour
		case "w":
			duration = time.Duration(num) * 7 * 24 * time.Hour
		default:
			return "", fmt.Errorf("unsupported unit: %s", unit)
		}

		targetDate := baseDate
		if sign == "-" {
			targetDate = targetDate.Add(-duration)
		} else {
			targetDate = targetDate.Add(duration)
		}

		return targetDate.Format("2006-01-02"), nil
	}

	// Handle already ISO format (YYYY-MM-DD)
	isoPattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if isoPattern.MatchString(input) {
		return input, nil
	}

	return "", fmt.Errorf("unsupported date format: %s", input)
}

// ConvertSearchQuery converts search query strings containing GitHub Projects date expressions
func ConvertSearchQuery(query string) (string, error) {
	// Pattern to match date field expressions: field:@today, field:>@today-1w, etc.
	pattern := regexp.MustCompile(`\b(\w+):([-<>=]*@\w+(?:[+-]\d+[dw])?)\b`)

	result := pattern.ReplaceAllStringFunc(query, func(match string) string {
		// Split the match into field:value
		parts := strings.SplitN(match, ":", 2)
		if len(parts) != 2 {
			return match // Return original if can't parse
		}

		field := parts[0]
		dateExpr := parts[1]

		// Convert the date expression
		converted, err := ConvertProjectsDateToISO(dateExpr)
		if err != nil {
			return match // Return original if conversion fails
		}

		return field + ":" + converted
	})

	return result, nil
}
