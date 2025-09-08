package domain_test

import (
	"testing"

	"github.com/HallyG/fingrab/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestToMajorUnit(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		money    domain.Money
		expected float64
	}{
		"returns float when nil currency": {
			money: domain.Money{
				MinorUnit: 1245,
			},
			expected: float64(1245),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected, test.money.ToMajorUnit())
		})
	}
}

func TestCurrencyFormat(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		money    domain.Money
		expected string
	}{
		"format GBP positive amount": {
			money: domain.Money{
				MinorUnit: 1245,
				Currency:  "GBP",
			},
			expected: "12.45",
		},
		"format: JPY positive amount": {
			money: domain.Money{
				MinorUnit: 1245,
				Currency:  "JPY",
			},
			expected: "1245",
		},
		"format GBP zero amount": {
			money: domain.Money{
				MinorUnit: 0,
				Currency:  "GBP",
			},
			expected: "0.00",
		},
		"format JPY zero amount": {
			money: domain.Money{
				MinorUnit: 0,
				Currency:  "JPY",
			},
			expected: "0",
		},
		"format GBP negative amount": {
			money: domain.Money{
				MinorUnit: -1245,
				Currency:  "GBP",
			},
			expected: "-12.45",
		},
		"format JPY negative amount": {
			money: domain.Money{
				MinorUnit: -1245,
				Currency:  "JPY",
			},
			expected: "-1245",
		},
		"format invalid currency": {
			money: domain.Money{
				MinorUnit: 1245,
				Currency:  "XYZ",
			},
			expected: "invalid currency: 1245 (XYZ)",
		},
		"format empty currency": {
			money: domain.Money{
				MinorUnit: 1245,
				Currency:  "",
			},
			expected: "invalid currency: 1245 ()",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected, test.money.String())
		})
	}
}
