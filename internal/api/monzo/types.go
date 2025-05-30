package monzo

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
)

type (
	UserID        string
	AccountID     string
	TransactionID string
	MerchantID    string
	PotID         string
)

type Owner struct {
	UserID             UserID `json:"user_id"`
	PreferredName      string `json:"preferred_name"`
	PreferredFirstName string `json:"preferred_first_name"`
}

type Account struct {
	ID                AccountID `json:"id"`
	Description       string    `json:"description"`
	CreatedAt         time.Time `json:"created"`
	Closed            bool      `json:"closed"`
	Currency          string    `json:"currency"`
	Type              string    `json:"type"`
	OwnerType         string    `json:"owner_type"`
	CountryCode       string    `json:"country_code"`
	CountryCodeAlpha3 string    `json:"country_code_alpha3"`
	AccountNumber     string    `json:"account_number"`
	SortCode          string    `json:"sort_code"`
	Owners            []*Owner  `json:"owners"`
}

type CounterParty struct {
	AccountNumber string `json:"account_number"`
	Name          string `json:"name"`
	SortCode      string `json:"sort_code"`
	UserID        UserID `json:"user_id"`
}

type Merchant struct {
	ID       MerchantID `json:"id"`
	Name     string     `json:"name"`
	Category string     `json:"category"`
	Online   bool       `json:"online"`
	Atm      bool       `json:"atm"`
}

type Pot struct {
	ID       PotID  `json:"id"`
	Name     string `json:"name"`
	Deleted  bool   `json:"deleted"`
	Currency string `json:"currency"`
}

type Transaction struct {
	ID              TransactionID `json:"id"`
	Description     string        `json:"description"`
	CreatedAt       time.Time     `json:"created"`
	Amount          domain.Money  `json:"amount"`
	UserNotes       string        `json:"notes"`
	CategoryName    string        `json:"category"`
	SettledAt       *time.Time    `json:"settled"`
	LocalAmount     domain.Money  `json:"local_money"`
	UpdatedAt       time.Time     `json:"updated"`
	AccountID       AccountID     `json:"account_id"`
	AmountIsPending bool          `json:"amount_is_pending"`
	Scheme          string        `json:"scheme"`
	Merchant        *Merchant     `json:"merchant"`
	CounterParty    *CounterParty `json:"counterparty"`
	DeclineReason   string        `json:"decline_reason"`
	Metadata        map[string]string
}

func (t *Transaction) UnmarshalJSON(data []byte) error {
	// Temporary struct to mirror original JSON structure
	type Alias Transaction

	temp := &struct {
		AmountMinorUnit      int64  `json:"amount"`
		Currency             string `json:"currency"`
		LocalAmountMinorUnit int64  `json:"local_amount"`
		LocalCurrency        string `json:"local_currency"`
		SettledAt            string `json:"settled"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}

	if err := json.Unmarshal(data, temp); err != nil {
		return err
	}

	if (*t.CounterParty == CounterParty{}) {
		t.CounterParty = nil
	}

	t.Amount = domain.Money{
		MinorUnit: temp.AmountMinorUnit,
		Currency:  temp.Currency,
	}

	t.LocalAmount = domain.Money{
		MinorUnit: temp.LocalAmountMinorUnit,
		Currency:  temp.LocalCurrency,
	}

	if temp.SettledAt != "" {
		time, err := time.Parse(time.RFC3339, temp.SettledAt)
		if err != nil {
			return err
		}

		t.SettledAt = &time
	}

	return nil
}

type Error struct {
	HTTPStatus int
	Code       string `json:"code"`
	Message    string `json:"message"`
}

/*
Error (statusCode=400, code=bad_request.bad_time_range, message=Error listing hydrated transactions by account)
Error (statusCode=400, code=bad_request.invalid_time_range, message=The time range you have requested is too large, please use the `since` and `before` parameters to request smaller ranges. Learn more in our community post: https://community.monzo.com/t/changes-when-listing-with-our-api/158676)
Error (statusCode=403, code=forbidden.verification_required, message=Verification required) when requesting old transactions.
*/
func (err Error) Error() string {
	return fmt.Sprintf("%s (http status=%d)", err.Message, err.HTTPStatus)
}

func UnmarshalError(status int, body []byte) error {
	if len(body) != 0 {
		apiError := Error{}
		if err := json.Unmarshal(body, &apiError); err == nil && apiError.Code != "" {
			apiError.HTTPStatus = status
		}

		return apiError
	}

	return Error{
		HTTPStatus: status,
		Code:       "unknown",
		Message:    "unknown",
	}
}
