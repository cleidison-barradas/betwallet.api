package domain

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestParse_ValoresValidos(t *testing.T) {
	cases := []struct {
		name     string
		amount   string
		currency Currency
		want     string
	}{
		{"valor inteiro simples", "25.00", "BRL", "25.00"},
		{"valor com centavos", "1000.50", "BRL", "1000.50"},
		{"zero explícito", "0.00", "BRL", "0.00"},
		{"negativo permitido no Parse genérico", "-25.00", "BRL", "-25.00"},
		{"um dígito antes do ponto", "5.00", "BRL", "5.00"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := Parse(tc.amount, tc.currency)
			if err != nil {
				t.Fatalf("esperava sucesso, obteve erro: %v", err)
			}
			if got := m.String(); got != tc.want {
				t.Errorf("String() = %q, esperado %q", got, tc.want)
			}
		})
	}
}

func TestParse_ValoresInvalidos(t *testing.T) {
	cases := []struct {
		name     string
		amount   string
		currency Currency
		wantErr  error
	}{
		{"vazio", "", "BRL", ErrInvalidAmount},
		{"sem casas decimais", "25", "BRL", ErrInvalidAmount},
		{"uma casa decimal", "25.0", "BRL", ErrInvalidAmount},
		{"escala excedente", "25.000", "BRL", ErrInvalidAmount},
		{"notação científica", "2.5e1", "BRL", ErrInvalidAmount},
		{"NaN", "NaN", "BRL", ErrInvalidAmount},
		{"Infinity", "Infinity", "BRL", ErrInvalidAmount},
		{"separador de milhar", "1,000.00", "BRL", ErrInvalidAmount},
		{"zero à esquerda", "025.00", "BRL", ErrInvalidAmount},
		{"letras no valor", "ab.cd", "BRL", ErrInvalidAmount},
		{"moeda com 2 letras", "25.00", "BR", ErrInvalidCurrency},
		{"moeda minúscula", "25.00", "brl", ErrInvalidCurrency},
		{"moeda vazia", "25.00", "", ErrInvalidCurrency},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.amount, tc.currency)
			if err == nil {
				t.Fatalf("esperava erro, obteve sucesso")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("erro = %v, esperava tipo %v", err, tc.wantErr)
			}
		})
	}
}

func TestParse_Overflow(t *testing.T) {
	_, err := Parse("999999999999999999999.00", "BRL")
	if !errors.Is(err, ErrAmountOverflow) {
		t.Errorf("esperava ErrAmountOverflow, obteve: %v", err)
	}
}

func TestAdd(t *testing.T) {
	a, _ := Parse("25.00", "BRL")
	b, _ := Parse("10.50", "BRL")

	sum, err := a.Add(b)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if sum.String() != "35.50" {
		t.Errorf("Add() = %s, esperado 35.50", sum.String())
	}
}

func TestSub_PodeGerarNegativo(t *testing.T) {
	a, _ := Parse("10.00", "BRL")
	b, _ := Parse("25.00", "BRL")

	diff, err := a.Sub(b)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if diff.String() != "-15.00" {
		t.Errorf("Sub() = %s, esperado -15.00", diff.String())
	}
	if !diff.IsNegative() {
		t.Error("esperava IsNegative() == true")
	}
}

func TestNegate(t *testing.T) {
	a, _ := Parse("25.00", "BRL")
	neg := a.Negate()

	if neg.String() != "-25.00" {
		t.Errorf("Negate() = %s, esperado -25.00", neg.String())
	}

	if a.String() != "25.00" {
		t.Errorf("valor original foi alterado: %s", a.String())
	}
}

func TestCmp(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want int
	}{
		{"a menor que b", "10.00", "25.00", -1},
		{"a maior que b", "25.00", "10.00", 1},
		{"iguais", "25.00", "25.00", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := Parse(tc.a, "BRL")
			b, _ := Parse(tc.b, "BRL")

			got, err := a.Cmp(b)
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if got != tc.want {
				t.Errorf("Cmp() = %d, esperado %d", got, tc.want)
			}
		})
	}
}

func TestJSON_RoundTrip(t *testing.T) {
	original, _ := Parse("25.00", "BRL")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("erro ao serializar: %v", err)
	}

	want := `{"amount":"25.00","currency":"BRL"}`
	if string(data) != want {
		t.Errorf("JSON = %s, esperado %s", data, want)
	}

	var decoded Money
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("erro ao desserializar: %v", err)
	}
	if decoded.String() != original.String() || decoded.Currency() != original.Currency() {
		t.Errorf("round-trip divergente: %+v vs %+v", decoded, original)
	}
}

func TestJSON_Invalido(t *testing.T) {
	var m Money
	err := json.Unmarshal([]byte(`{"amount":"25","currency":"BRL"}`), &m)
	if !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("esperava ErrInvalidAmount, obteve: %v", err)
	}
}
