package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func calcPayment(payment string, percentage string) (int, error) {
	p, err := dollarStringToCents(payment)
	if err != nil {
		return 0, fmt.Errorf("%w %w", ErrAmount, err)
	}
	perc, err := convertPercToFloat(percentage)
	if err != nil {
		return 0, fmt.Errorf("%w %w", ErrPercentage, err)
	}
	//partPay := p
	if perc <= 0 {
		return 0, fmt.Errorf("%w %w", ErrPercentage, fmt.Errorf("percentage value must be greater than 0"))
	}
	partPay := float64(p) * perc / 100
	return int(math.Round(partPay)), nil
}

func convertToFloat(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("field is blank")
	}
	factor := 1.0
	if value[0] == '(' {
		factor = -1.0
		value = strings.ReplaceAll(value, "(", "")
		value = strings.ReplaceAll(value, ")", "")
	}
	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("error converting value")
	}
	return num * factor, nil
}

func convertPercToFloat(value string) (float64, error) {
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
	return int(value * 100), nil
}
