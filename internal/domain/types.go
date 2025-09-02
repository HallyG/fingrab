package domain

import (
	"fmt"
	"math"
	"time"

	"github.com/Rhymond/go-money"
)

type Money struct {
	MinorUnit int64  `json:"minorUnits"`
	Currency  string `json:"currency"` // ISO4217 Alpha Currency code
}

func (m Money) ToMajorUnit() float64 {
	currency := money.GetCurrency(m.Currency)
	if currency == nil {
		return float64(m.MinorUnit)
	}

	return float64(m.MinorUnit) / math.Pow10(currency.Fraction)
}

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
	IsDeposit bool
	BankName  string
	Notes     string
}
