package starling

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/util/uuidutil"
	"github.com/google/uuid"
)

type (
	AccountID      uuid.UUID
	FeedItemID     uuid.UUID
	CategoryID     uuid.UUID
	CounterPartyID uuid.UUID
	SavingsGoalID  uuid.UUID
)

func (a *AccountID) UnmarshalJSON(data []byte) error {
	id, err := uuidutil.UnmarshallJSONUUID(data)
	if err != nil {
		return err
	}

	*a = AccountID(id)

	return nil
}

func (a AccountID) String() string {
	return uuid.UUID(a).String()
}

func (f *FeedItemID) UnmarshalJSON(data []byte) error {
	id, err := uuidutil.UnmarshallJSONUUID(data)
	if err != nil {
		return err
	}

	*f = FeedItemID(id)

	return nil
}

func (f FeedItemID) String() string {
	return uuid.UUID(f).String()
}

func (s *SavingsGoalID) UnmarshalJSON(data []byte) error {
	id, err := uuidutil.UnmarshallJSONUUID(data)
	if err != nil {
		return err
	}

	*s = SavingsGoalID(id)

	return nil
}

func (s SavingsGoalID) String() string {
	return uuid.UUID(s).String()
}

func (c *CategoryID) UnmarshalJSON(data []byte) error {
	id, err := uuidutil.UnmarshallJSONUUID(data)
	if err != nil {
		return err
	}

	*c = CategoryID(id)

	return nil
}

func (c CategoryID) String() string {
	return uuid.UUID(c).String()
}

func (c *CounterPartyID) UnmarshalJSON(data []byte) error {
	id, err := uuidutil.UnmarshallJSONUUID(data)
	if err != nil {
		return err
	}

	*c = CounterPartyID(id)

	return nil
}

func (c CounterPartyID) String() string {
	return uuid.UUID(c).String()
}

type Account struct {
	ID                AccountID  `json:"accountUid"`
	Type              string     `json:"accountType"`
	DefaultCategoryID CategoryID `json:"defaultCategory"`
	Currency          string     `json:"currency"`
	CreatedAt         time.Time  `json:"createdAt"`
	Name              string     `json:"name"`
}

type SavingsGoal struct {
	ID         SavingsGoalID `json:"savingsGoalUid"`
	Name       string        `json:"name"`
	State      string        `json:"state"`
	Target     domain.Money  `json:"target"`
	TotalSaved domain.Money  `json:"totalSaved"`
}

type (
	Direction string
	Status    string
)

const (
	DirectionIN             Direction = "IN"
	DirectionOUT            Direction = "OUT"
	StatusUpcoming          Status    = "UPCOMING"
	StatusUpcomingCancelled Status    = "UPCOMING_CANCELLED"
	StatusPending           Status    = "PENDING"
	StatusReversed          Status    = "REVERSED"
	StatusSettled           Status    = "SETTLED"
	StatusDeclined          Status    = "DECLINED"
	StatusRefunded          Status    = "REFUNDED"
	StatusRetrying          Status    = "RETRYING"
	StatusAccountCheck      Status    = "ACCOUNT_CHECK"
)

type RoundUp struct {
	GoalCategoryID CategoryID   `json:"goalCategoryUid"`
	Amount         domain.Money `json:"amount"`
}

type FeedItem struct {
	ID                      FeedItemID     `json:"feedItemUid"`
	Amount                  domain.Money   `json:"amount"` // Amount in the account's currency
	TransactedAt            time.Time      `json:"transactionTime"`
	SettledAt               *time.Time     `json:"settlementTime"`
	CategoryID              CategoryID     `json:"categoryUid"`
	CategoryName            string         `json:"spendingCategory"`
	Description             string         `json:"reference"`
	Status                  Status         `json:"status"`
	UserNote                string         `json:"userNote"`
	Direction               Direction      `json:"direction"`        // Direction of payment, e.g. IN or OUT
	Source                  string         `json:"source"`           // e.g. MASTED_CARD
	SourceSubType           string         `json:"sourceSubType"`    // e.g. Online, ATM, Deposit
	CounterPartyType        string         `json:"counterPartyType"` // e.g. STARLING, MERCHANT
	CounterPartyID          CounterPartyID `json:"counterPartyUid"`
	CounterPartySubEntityID string         `json:"counterPartySubEntityUid"`
	CounterPartyName        string         `json:"counterPartyName"`
	RoundUp                 *RoundUp       `json:"roundUp"`
}

type ErrorMessage struct {
	Message string `json:"message"`
}

type Error struct {
	HTTPStatus    int
	Code          string         `json:"error"`
	Message       string         `json:"error_description"`
	Success       bool           `json:"success"`
	ErrorMessages []ErrorMessage `json:"errors"`
}

func (err Error) Error() string {
	var sb strings.Builder

	if err.Message != "" {
		sb.WriteString(err.Message)
		sb.WriteString(" ")
	}

	if len(err.ErrorMessages) > 0 {
		errorMessages := make([]string, len(err.ErrorMessages))
		for i, e := range err.ErrorMessages {
			errorMessages[i] = e.Message
		}

		sb.WriteString("[")
		sb.WriteString(strings.Join(errorMessages, ", "))
		sb.WriteString("] ")
	}

	if err.HTTPStatus != 0 {
		errorMessages := make([]string, len(err.ErrorMessages))
		for i, e := range err.ErrorMessages {
			errorMessages[i] = e.Message
		}

		sb.WriteString("(http status=")
		sb.WriteString(strconv.Itoa(err.HTTPStatus))
		sb.WriteString(")")
	}

	return sb.String()
}

func UnmarshalError(status int, body []byte) error {
	if len(body) != 0 {
		apiError := Error{}
		if err := json.Unmarshal(body, &apiError); err == nil {
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
