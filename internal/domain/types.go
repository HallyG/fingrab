package domain

import (
	"fmt"
	"math"
	"time"

	"github.com/Rhymond/go-money"
)

// Money represents a monetary amount in a specific currency, stored in minor units (e.g. pennies for GBP).
type Money struct {
	MinorUnit int64  `json:"minorUnits"` // Amount in the currency's smallest unit (e.g. 100 for £1.00 GBP)
	Currency  string `json:"currency"`   // ISO4217 Alpha Currency code (e.g. USD, EUR, GBP)
}

// ToMajorUnit converts the Money amount from minor units to major units (e.g., cents to dollars).
// If the currency is invalid or not found, it returns the minor unit as a float64 without conversion.
// Example:
//
//	m := Money{MinorUnit: 10050, Currency: "GBP"}
//	major := m.ToMajorUnit() // Returns 100.50
func (m Money) ToMajorUnit() float64 {
	currency := money.GetCurrency(m.Currency)
	if currency == nil {
		return float64(m.MinorUnit)
	}

	return float64(m.MinorUnit) / math.Pow10(currency.Fraction)
}

// String returns a human-readable string representation of the Money amount in major units with the currency's fractional precision.
// If the currency is invalid, it returns a string indicating the error along with the raw minor unit and currency code.
//
// Example:
//
//	m := Money{MinorUnit: 10050, Currency: "GBP"}
//	fmt.Println(m.String()) // Outputs: "100.50"
//	m := Money{MinorUnit: 10050, Currency: "JPY"}
//	fmt.Println(m.String()) // Outputs: "10050"
//	m = Money{MinorUnit: 10050, Currency: "INVALID"}
//	fmt.Println(m.String()) // Outputs: "invalid currency: 10050 (INVALID)
func (m Money) String() string {
	currency := money.GetCurrency(m.Currency)
	if currency == nil {
		return fmt.Sprintf("invalid currency: %d (%s)", m.MinorUnit, m.Currency)
	}

	return fmt.Sprintf("%.*f", currency.Fraction, m.ToMajorUnit())
}

type Transaction struct {
	Amount    Money
	Reference string
	Category  string
	CreatedAt time.Time
	IsDeposit bool   // Indicates if the transaction is a deposit (true) or withdrawal (false)
	BankName  string // The name of the bank the transaction was exported from.
	Notes     string
}
