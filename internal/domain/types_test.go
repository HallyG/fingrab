package domain_test

import (
	"testing"

	"github.com/HallyG/fingrab/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestCurrencyFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		money    domain.Money
		expected string
	}{
		{
			name: "GBP positive amount",
			money: domain.Money{
				MinorUnit: 1245,
				Currency:  "GBP",
			},
			expected: "12.45",
		},
		{
			name: "JPY positive amount",
			money: domain.Money{
				MinorUnit: 1245,
				Currency:  "JPY",
			},
			expected: "1245",
		},
		{
			name: "GBP zero amount",
			money: domain.Money{
				MinorUnit: 0,
				Currency:  "GBP",
			},
			expected: "0.00",
		},
		{
			name: "JPY zero amount",
			money: domain.Money{
				MinorUnit: 0,
				Currency:  "JPY",
			},
			expected: "0",
		},
		{
			name: "GBP negative amount",
			money: domain.Money{
				MinorUnit: -1245,
				Currency:  "GBP",
			},
			expected: "-12.45",
		},
		{
			name: "JPY negative amount",
			money: domain.Money{
				MinorUnit: -1245,
				Currency:  "JPY",
			},
			expected: "-1245",
		},
		{
			name: "Invalid currency",
			money: domain.Money{
				MinorUnit: 1245,
				Currency:  "XYZ",
			},
			expected: "invalid currency: 1245 (XYZ)",
		},
		{
			name: "Empty currency",
			money: domain.Money{
				MinorUnit: 1245,
				Currency:  "",
			},
			expected: "invalid currency: 1245 ()",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected, test.money.String())
		})
	}
}
