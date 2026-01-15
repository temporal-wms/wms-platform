package domain

// AccountType represents the type of ledger account in double-entry bookkeeping
type AccountType string

const (
	// AccountInventory - Asset account representing physical goods on hand
	AccountInventory AccountType = "INVENTORY"

	// AccountCOGS - Expense account representing cost of goods sold
	AccountCOGS AccountType = "COGS"

	// AccountGoodsInTransit - Asset account for received goods not yet put away
	AccountGoodsInTransit AccountType = "GOODS_IN_TRANSIT"

	// AccountAdjustments - Contra-asset account for shrinkage, damage, cycle count corrections
	AccountAdjustments AccountType = "ADJUSTMENTS"

	// AccountReturns - Asset account for returned goods pending processing
	AccountReturns AccountType = "RETURNS"
)

// IsValid checks if the account type is valid
func (a AccountType) IsValid() bool {
	switch a {
	case AccountInventory, AccountCOGS, AccountGoodsInTransit, AccountAdjustments, AccountReturns:
		return true
	default:
		return false
	}
}

// String returns the string representation of the account type
func (a AccountType) String() string {
	return string(a)
}

// IsAsset returns true if this is an asset account
func (a AccountType) IsAsset() bool {
	switch a {
	case AccountInventory, AccountGoodsInTransit, AccountReturns:
		return true
	default:
		return false
	}
}

// IsExpense returns true if this is an expense account
func (a AccountType) IsExpense() bool {
	return a == AccountCOGS
}

// IsContraAsset returns true if this is a contra-asset account
func (a AccountType) IsContraAsset() bool {
	return a == AccountAdjustments
}

// AccountBalance represents the balance of a specific account
type AccountBalance struct {
	Balance int   `bson:"balance" json:"balance"` // Quantity balance
	Value   Money `bson:"value" json:"value"`     // Monetary value
}

// NewAccountBalance creates a new account balance
func NewAccountBalance(balance int, value Money) AccountBalance {
	return AccountBalance{
		Balance: balance,
		Value:   value,
	}
}

// ZeroAccountBalance creates a zero balance for an account
func ZeroAccountBalance(currency string) AccountBalance {
	return AccountBalance{
		Balance: 0,
		Value:   ZeroMoney(currency),
	}
}

// IsZero returns true if the balance is zero
func (ab AccountBalance) IsZero() bool {
	return ab.Balance == 0 && ab.Value.IsZero()
}
