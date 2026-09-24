package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Currency string

const (
	USD Currency = "USD"
	EUR Currency = "EUR"
	BRL Currency = "BRL"
)

var (
	ErrInvalidCurrency = errors.New("invalid currency")
	ErrInvalidAmount   = errors.New("invalid amount")
	ErrAmountOverflow  = errors.New("amount overflow")
)

func newCurrency(code Currency) (Currency, error) {
	if len(code) != 3 {
		return "", fmt.Errorf("%w: %q", ErrInvalidCurrency, code)
	}

	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return "", fmt.Errorf("%w: %q", ErrInvalidCurrency, code)
		}
	}

	return Currency(code), nil
}

type Money struct {
	minorUnits int64
	currency   Currency
}

func Zero(currencyCode Currency) (Money, error) {
	cur, err := newCurrency(currencyCode)
	if err != nil {
		return Money{}, fmt.Errorf("%w: %q", ErrInvalidCurrency, currencyCode)
	}
	return Money{
		minorUnits: 0,
		currency:   cur,
	}, nil
}

func Parse(amount string, currencyCode Currency) (Money, error) {
	cur, err := newCurrency(currencyCode)
	if err != nil {
		return Money{}, err
	}

	minorUnits, err := parseDecimalToMinorUnits(amount)
	if err != nil {
		return Money{}, err
	}

	return Money{
		minorUnits: minorUnits,
		currency:   cur,
	}, nil
}

func parseDecimalToMinorUnits(amount string) (int64, error) {
	if amount == "" {
		return 0, fmt.Errorf("%w: empty amount", ErrInvalidAmount)
	}

	negative := false
	s := amount

	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	dotIndex := strings.IndexByte(s, '.')
	if dotIndex == -1 {
		return 0, fmt.Errorf("%w: required two decimal places (\"25.00\")", ErrInvalidAmount)
	}

	intPart := s[:dotIndex]
	fracPart := s[dotIndex+1:]

	if intPart == "" || len(fracPart) != 2 {
		return 0, fmt.Errorf("%w: required two decimal places (\"25.00\")", ErrInvalidAmount)
	}

	if len(intPart) > 1 && intPart[0] == '0' {
		return 0, fmt.Errorf("%w: zero is not allowed", ErrInvalidAmount)
	}

	if !onlyDigits(intPart) || !onlyDigits(fracPart) {
		return 0, fmt.Errorf("%w: only digits are allowed", ErrInvalidAmount)
	}

	intValue, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrAmountOverflow, err)
	}

	fracValue, err := strconv.ParseInt(fracPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrAmountOverflow, err)
	}

	if intValue > (math.MaxInt64-fracValue)/100 {
		return 0, fmt.Errorf("%w: %v", ErrAmountOverflow, err)
	}

	minorUnits := intValue*100 + fracValue

	if negative {
		minorUnits = -minorUnits
	}

	return minorUnits, nil
}

func onlyDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (m Money) Currency() Currency {
	return m.currency
}

func (m Money) IsZero() bool {
	return m.minorUnits == 0
}

func (m Money) IsNegative() bool {
	return m.minorUnits < 0
}

func (m Money) IsPositive() bool {
	return m.minorUnits > 0
}

func (s Money) sameCurrency(other Money) error {
	if s.currency != other.currency {
		return ErrInvalidCurrency
	}
	return nil
}

func (m Money) Add(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	sum, ok := addOverflowSafe(m.minorUnits, other.minorUnits)
	if !ok {
		return Money{}, ErrAmountOverflow
	}
	return Money{
		minorUnits: sum,
		currency:   m.currency,
	}, nil
}

func (m Money) Sub(other Money) (Money, error) {
	return m.Add(other.Negate())
}

func (m Money) Negate() Money {
	if m.minorUnits == math.MinInt64 {
		panic("money: overflow negate")
	}
	return Money{minorUnits: -m.minorUnits, currency: m.currency}
}

func addOverflowSafe(a, b int64) (int64, bool) {
	sum := a + b
	if (a > 0 && b > 0 && sum < 0) || (a < 0 && b < 0 && sum > 0) {
		return 0, false
	}
	return sum, true
}

func (m Money) Cmp(other Money) (int, error) {
	if err := m.sameCurrency(other); err != nil {
		return 0, err
	}
	switch {
	case m.minorUnits < other.minorUnits:
		return -1, nil
	case m.minorUnits > other.minorUnits:
		return 1, nil
	default:
		return 0, nil
	}
}

func (m Money) String() string {
	sign := ""
	units := m.minorUnits
	if units < 0 {
		sign = "-"
		units = -units
	}
	return fmt.Sprintf("%s%d.%02d", sign, units/100, units%100)
}

type moneyJSON struct {
	Amount   string   `json:"amount"`
	Currency Currency `json:"currency"`
}

func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(moneyJSON{Amount: m.String(), Currency: m.currency})
}

func (m *Money) UnmarshalJSON(data []byte) error {
	var raw moneyJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidAmount, err)
	}
	parsed, err := Parse(raw.Amount, raw.Currency)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

func (m Money) MinorUnits() int64 { return m.minorUnits }

func FromMinorUnits(units int64, currency Currency) (Money, error) {
	cur, err := newCurrency(currency)
	if err != nil {
		return Money{}, err
	}

	return Money{
		minorUnits: units,
		currency:   cur,
	}, nil
}
