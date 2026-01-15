package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

// LedgerEntryID represents a unique identifier for a ledger entry
type LedgerEntryID struct {
	value string
}

// NewLedgerEntryID creates a new unique ledger entry ID
func NewLedgerEntryID() LedgerEntryID {
	timestamp := time.Now().UTC().Format("20060102150405")
	return LedgerEntryID{
		value: fmt.Sprintf("LE-%s-%s", timestamp, uuid.New().String()[:8]),
	}
}

// ParseLedgerEntryID parses a string into a LedgerEntryID
func ParseLedgerEntryID(s string) (LedgerEntryID, error) {
	if s == "" {
		return LedgerEntryID{}, errors.New("ledger entry ID cannot be empty")
	}
	return LedgerEntryID{value: s}, nil
}

// String returns the string representation
func (id LedgerEntryID) String() string {
	return id.value
}

// MarshalBSONValue implements bson.ValueMarshaler
func (id LedgerEntryID) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.MarshalValue(id.value)
}

// UnmarshalBSONValue implements bson.ValueUnmarshaler
func (id *LedgerEntryID) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	return bson.UnmarshalValue(t, data, &id.value)
}

// LedgerTransactionID represents a unique identifier linking debit/credit entry pairs
type LedgerTransactionID struct {
	value string
}

// NewLedgerTransactionID creates a new unique transaction ID
func NewLedgerTransactionID() LedgerTransactionID {
	timestamp := time.Now().UTC().Format("20060102150405")
	return LedgerTransactionID{
		value: fmt.Sprintf("LTXN-%s-%s", timestamp, uuid.New().String()[:8]),
	}
}

// ParseLedgerTransactionID parses a string into a LedgerTransactionID
func ParseLedgerTransactionID(s string) (LedgerTransactionID, error) {
	if s == "" {
		return LedgerTransactionID{}, errors.New("ledger transaction ID cannot be empty")
	}
	return LedgerTransactionID{value: s}, nil
}

// String returns the string representation
func (id LedgerTransactionID) String() string {
	return id.value
}

// MarshalBSONValue implements bson.ValueMarshaler
func (id LedgerTransactionID) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.MarshalValue(id.value)
}

// UnmarshalBSONValue implements bson.ValueUnmarshaler
func (id *LedgerTransactionID) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	return bson.UnmarshalValue(t, data, &id.value)
}

// LedgerEntry represents a single entry in the ledger (debit or credit)
// In double-entry bookkeeping, each transaction creates at least 2 entries
type LedgerEntry struct {
	EntryID        LedgerEntryID       `bson:"entryId" json:"entryId"`
	TransactionID  LedgerTransactionID `bson:"transactionId" json:"transactionId"` // Links debit/credit pair
	AccountType    AccountType         `bson:"accountType" json:"accountType"`
	DebitAmount    int                 `bson:"debitAmount" json:"debitAmount"`       // Positive for debit
	CreditAmount   int                 `bson:"creditAmount" json:"creditAmount"`     // Positive for credit
	DebitValue     Money               `bson:"debitValue" json:"debitValue"`         // Cost value for debit
	CreditValue    Money               `bson:"creditValue" json:"creditValue"`       // Cost value for credit
	RunningBalance int                 `bson:"runningBalance" json:"runningBalance"` // Balance AFTER this entry
	RunningValue   Money               `bson:"runningValue" json:"runningValue"`     // Value AFTER this entry
	SKU            string              `bson:"sku" json:"sku"`
	LocationID     string              `bson:"locationId" json:"locationId"`
	UnitCost       Money               `bson:"unitCost" json:"unitCost"`
	ReferenceID    string              `bson:"referenceId" json:"referenceId"`     // Order ID, PO ID, etc.
	ReferenceType  string              `bson:"referenceType" json:"referenceType"` // "order", "po", "adjustment", "transfer"
	Description    string              `bson:"description" json:"description"`
	CreatedAt      time.Time           `bson:"createdAt" json:"createdAt"`
	CreatedBy      string              `bson:"createdBy" json:"createdBy"`
}

// NewDebitEntry creates a debit entry
func NewDebitEntry(
	transactionID LedgerTransactionID,
	accountType AccountType,
	amount int,
	unitCost Money,
	runningBalance int,
	runningValue Money,
	sku, locationID, referenceID, referenceType, description, createdBy string,
) (LedgerEntry, error) {
	if amount <= 0 {
		return LedgerEntry{}, errors.New("debit amount must be positive")
	}

	debitValue, err := unitCost.Multiply(amount)
	if err != nil {
		return LedgerEntry{}, err
	}

	return LedgerEntry{
		EntryID:        NewLedgerEntryID(),
		TransactionID:  transactionID,
		AccountType:    accountType,
		DebitAmount:    amount,
		CreditAmount:   0,
		DebitValue:     debitValue,
		CreditValue:    ZeroMoney(unitCost.Currency()),
		RunningBalance: runningBalance,
		RunningValue:   runningValue,
		SKU:            sku,
		LocationID:     locationID,
		UnitCost:       unitCost,
		ReferenceID:    referenceID,
		ReferenceType:  referenceType,
		Description:    description,
		CreatedAt:      time.Now().UTC(),
		CreatedBy:      createdBy,
	}, nil
}

// NewCreditEntry creates a credit entry
func NewCreditEntry(
	transactionID LedgerTransactionID,
	accountType AccountType,
	amount int,
	unitCost Money,
	runningBalance int,
	runningValue Money,
	sku, locationID, referenceID, referenceType, description, createdBy string,
) (LedgerEntry, error) {
	if amount <= 0 {
		return LedgerEntry{}, errors.New("credit amount must be positive")
	}

	creditValue, err := unitCost.Multiply(amount)
	if err != nil {
		return LedgerEntry{}, err
	}

	return LedgerEntry{
		EntryID:        NewLedgerEntryID(),
		TransactionID:  transactionID,
		AccountType:    accountType,
		DebitAmount:    0,
		CreditAmount:   amount,
		DebitValue:     ZeroMoney(unitCost.Currency()),
		CreditValue:    creditValue,
		RunningBalance: runningBalance,
		RunningValue:   runningValue,
		SKU:            sku,
		LocationID:     locationID,
		UnitCost:       unitCost,
		ReferenceID:    referenceID,
		ReferenceType:  referenceType,
		Description:    description,
		CreatedAt:      time.Now().UTC(),
		CreatedBy:      createdBy,
	}, nil
}

// IsDebit returns true if this is a debit entry
func (e LedgerEntry) IsDebit() bool {
	return e.DebitAmount > 0
}

// IsCredit returns true if this is a credit entry
func (e LedgerEntry) IsCredit() bool {
	return e.CreditAmount > 0
}

// Amount returns the absolute amount (debit or credit)
func (e LedgerEntry) Amount() int {
	if e.IsDebit() {
		return e.DebitAmount
	}
	return e.CreditAmount
}

// Value returns the absolute value (debit or credit)
func (e LedgerEntry) Value() Money {
	if e.IsDebit() {
		return e.DebitValue
	}
	return e.CreditValue
}
