package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func pct(v int) string {
	if v < 0 {
		return fmt.Sprintf("(%d.%02d)", -v/100, -v%100)
	}
	return fmt.Sprintf("%d.%02d", v/100, v%100)
}

func calcPayment(payment string, gst string, percentage string) (int, int, error) {
	p, err := dollarStringToCents(payment)
	if err != nil {
		return 0, 0, fmt.Errorf("%w %w", ErrAmount, err)
	}
	g, err := dollarStringToCents(gst)
	if err != nil {
		return 0, 0, fmt.Errorf("%w %w", ErrAmount, err)
	}
	perc, err := convertPercToFloat(percentage)
	if err != nil {
		return 0, 0, fmt.Errorf("%w %w", ErrPercentage, err)
	}
	//partPay := p
	if perc <= 0 {
		return 0, 0, fmt.Errorf("%w %w", ErrPercentage, fmt.Errorf("percentage value must be greater than 0"))
	}
	partPay := float64(p*perc) / 10000
	return int(math.Round(partPay)), p + g, nil
}

func convertToFloat(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("field is blank")
	}
	factor := 1
	if value[0] == '(' {
		factor = -1
		value = strings.ReplaceAll(value, "(", "")
		value = strings.ReplaceAll(value, ")", "")
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid number format")
	}

	integerPart := parts[0]
	fractionalPart := "00"
	if len(parts) == 2 {
		fractionalPart = parts[1]
		if len(fractionalPart) > 2 {
			fractionalPart = fractionalPart[:2]
		} else if len(fractionalPart) < 2 {
			fractionalPart = fractionalPart + "0"
		}
	}

	combined := integerPart + fractionalPart
	num, err := strconv.Atoi(combined)
	if err != nil {
		return 0, fmt.Errorf("error converting value")
	}
	return num * factor, nil
}

func convertPercToFloat(value string) (int, error) {
	value = strings.ReplaceAll(value, "%", "")
	return convertToFloat(value)
}

func dollarStringToCents(dollarStr string) (int, error) {
	// Remove any leading "$" sign if present
	dollarStr = strings.TrimPrefix(dollarStr, "$")

	// Parse the string to a float64
	value, err := convertToFloat(dollarStr)
	if err != nil {
		return 0, fmt.Errorf("invalid dollar amount: %v", err)
	}

	// convert to cents
	return value, nil
}
