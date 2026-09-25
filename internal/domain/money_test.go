package domain

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestParse_ValidValues(t *testing.T) {
	cases := []struct {
		name     string
		amount   string
		currency Currency
		want     string
	}{
		{"simple integer value", "25.00", "BRL", "25.00"},
		{"value with cents", "1000.50", "BRL", "1000.50"},
		{"explicit zero", "0.00", "BRL", "0.00"},
		{"negative allowed in generic Parse", "-25.00", "BRL", "-25.00"},
		{"single digit before the dot", "5.00", "BRL", "5.00"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := Parse(tc.amount, tc.currency)
			if err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}
			if got := m.String(); got != tc.want {
				t.Errorf("String() = %q, expected %q", got, tc.want)
			}
		})
	}
}

func TestParse_InvalidValues(t *testing.T) {
	cases := []struct {
		name     string
		amount   string
		currency Currency
		wantErr  error
	}{
		{"empty", "", "BRL", ErrInvalidAmount},
		{"no decimal places", "25", "BRL", ErrInvalidAmount},
		{"one decimal place", "25.0", "BRL", ErrInvalidAmount},
		{"excess scale", "25.000", "BRL", ErrInvalidAmount},
		{"scientific notation", "2.5e1", "BRL", ErrInvalidAmount},
		{"NaN", "NaN", "BRL", ErrInvalidAmount},
		{"Infinity", "Infinity", "BRL", ErrInvalidAmount},
		{"thousands separator", "1,000.00", "BRL", ErrInvalidAmount},
		{"leading zero", "025.00", "BRL", ErrInvalidAmount},
		{"letters in the value", "ab.cd", "BRL", ErrInvalidAmount},
		{"2-letter currency", "25.00", "BR", ErrInvalidCurrency},
		{"lowercase currency", "25.00", "brl", ErrInvalidCurrency},
		{"empty currency", "25.00", "", ErrInvalidCurrency},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.amount, tc.currency)
			if err == nil {
				t.Fatalf("expected error, got success")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("error = %v, expected type %v", err, tc.wantErr)
			}
		})
	}
}

func TestParse_Overflow(t *testing.T) {
	_, err := Parse("999999999999999999999.00", "BRL")
	if !errors.Is(err, ErrAmountOverflow) {
		t.Errorf("expected ErrAmountOverflow, got: %v", err)
	}
}

func TestAdd(t *testing.T) {
	a, _ := Parse("25.00", "BRL")
	b, _ := Parse("10.50", "BRL")

	sum, err := a.Add(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.String() != "35.50" {
		t.Errorf("Add() = %s, expected 35.50", sum.String())
	}
}

func TestSub_CanProduceNegative(t *testing.T) {
	a, _ := Parse("10.00", "BRL")
	b, _ := Parse("25.00", "BRL")

	diff, err := a.Sub(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff.String() != "-15.00" {
		t.Errorf("Sub() = %s, expected -15.00", diff.String())
	}
	if !diff.IsNegative() {
		t.Error("expected IsNegative() == true")
	}
}

func TestNegate(t *testing.T) {
	a, _ := Parse("25.00", "BRL")
	neg := a.Negate()

	if neg.String() != "-25.00" {
		t.Errorf("Negate() = %s, expected -25.00", neg.String())
	}

	if a.String() != "25.00" {
		t.Errorf("original value was mutated: %s", a.String())
	}
}

func TestCmp(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want int
	}{
		{"a less than b", "10.00", "25.00", -1},
		{"a greater than b", "25.00", "10.00", 1},
		{"equal", "25.00", "25.00", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := Parse(tc.a, "BRL")
			b, _ := Parse(tc.b, "BRL")

			got, err := a.Cmp(b)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("Cmp() = %d, expected %d", got, tc.want)
			}
		})
	}
}

func TestJSON_RoundTrip(t *testing.T) {
	original, _ := Parse("25.00", "BRL")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	want := `{"amount":"25.00","currency":"BRL"}`
	if string(data) != want {
		t.Errorf("JSON = %s, expected %s", data, want)
	}

	var decoded Money
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.String() != original.String() || decoded.Currency() != original.Currency() {
		t.Errorf("round-trip mismatch: %+v vs %+v", decoded, original)
	}
}

func TestJSON_Invalid(t *testing.T) {
	var m Money
	err := json.Unmarshal([]byte(`{"amount":"25","currency":"BRL"}`), &m)
	if !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("expected ErrInvalidAmount, got: %v", err)
	}
}
