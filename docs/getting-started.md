# Getting Started with Event Sourcing

This guide will walk you through building your first event-sourced application using the `es` library.

## Prerequisites

- Go 1.21 or later
- Basic understanding of event sourcing concepts
- Familiarity with Go interfaces and structs

## Installation

```bash
go mod init myapp
go get github.com/fgrzl/es
```

## Step 1: Define Your Domain Events

Start by defining the events that represent state changes in your domain:

```go
package bankaccount

import "github.com/fgrzl/es"

// Register events for polymorphic serialization
func init() {
    es.Register(func() *AccountOpened { return &AccountOpened{} })
    es.Register(func() *MoneyDeposited { return &MoneyDeposited{} })
    es.Register(func() *MoneyWithdrawn { return &MoneyWithdrawn{} })
}

type AccountOpened struct {
    es.DomainEventBase
    AccountID string `json:"account_id"`
    InitialBalance int64 `json:"initial_balance"`
}

func (e *AccountOpened) GetDiscriminator() string { return "account.opened" }
func (e *AccountOpened) GetSpaces() []string      { return []string{"bank-accounts"} }

type MoneyDeposited struct {
    es.DomainEventBase
    Amount int64 `json:"amount"`
}

func (e *MoneyDeposited) GetDiscriminator() string { return "money.deposited" }
func (e *MoneyDeposited) GetSpaces() []string      { return []string{"bank-accounts"} }

type MoneyWithdrawn struct {
    es.DomainEventBase
    Amount int64 `json:"amount"`
}

func (e *MoneyWithdrawn) GetDiscriminator() string { return "money.withdrawn" }
func (e *MoneyWithdrawn) GetSpaces() []string      { return []string{"bank-accounts"} }
```

## Step 2: Create Your Aggregate

Define an aggregate that encapsulates your business logic:

```go
package bankaccount

import (
    "context"
    "errors"
    "github.com/fgrzl/es"
    "github.com/google/uuid"
)

type BankAccount struct {
    es.Aggregate
    id        uuid.UUID
    accountID string
    balance   int64
    isOpen    bool
}

func NewBankAccount(id uuid.UUID, accountID string) *BankAccount {
    account := &BankAccount{
        Aggregate: es.NewAggregate(context.Background(), "bank-accounts", id),
        id:        id,
        accountID: accountID,
    }
    
    // Register event handlers
    es.RegisterHandler(account, account.OnAccountOpened)
    es.RegisterHandler(account, account.OnMoneyDeposited)
    es.RegisterHandler(account, account.OnMoneyWithdrawn)
    
    return account
}

// Business methods
func (a *BankAccount) Open(initialBalance int64) error {
    if a.isOpen {
        return errors.New("account is already open")
    }
    
    return a.Raise(&AccountOpened{
        AccountID: a.accountID,
        InitialBalance: initialBalance,
    })
}

func (a *BankAccount) Deposit(amount int64) error {
    if !a.isOpen {
        return errors.New("account is not open")
    }
    
    if amount <= 0 {
        return errors.New("amount must be positive")
    }
    
    return a.Raise(&MoneyDeposited{Amount: amount})
}

func (a *BankAccount) Withdraw(amount int64) error {
    if !a.isOpen {
        return errors.New("account is not open")
    }
    
    if amount <= 0 {
        return errors.New("amount must be positive")
    }
    
    if a.balance < amount {
        return errors.New("insufficient funds")
    }
    
    return a.Raise(&MoneyWithdrawn{Amount: amount})
}

// Event handlers
func (a *BankAccount) OnAccountOpened(e *AccountOpened) {
    a.isOpen = true
    a.balance = e.InitialBalance
}

func (a *BankAccount) OnMoneyDeposited(e *MoneyDeposited) {
    a.balance += e.Amount
}

func (a *BankAccount) OnMoneyWithdrawn(e *MoneyWithdrawn) {
    a.balance -= e.Amount
}

// Query methods
func (a *BankAccount) GetBalance() int64 {
    return a.balance
}

func (a *BankAccount) IsOpen() bool {
    return a.isOpen
}
```

Aggregate wiring is intentionally fail-fast. `NewAggregate`, `NewTenantAggregate`, `RegisterHandler`, and invalid event-area mappings in the default `Raise` implementation panic immediately when the aggregate definition is invalid. Treat those as programmer errors in aggregate design. Business-rule failures, such as trying to withdraw too much money, should still be returned as normal `error` values from command methods.

## Step 3: Use the Repository

Create a repository to persist and load your aggregates:

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/fgrzl/es"
    "github.com/google/uuid"
    "myapp/bankaccount"
)

func main() {
    // Create event store and repository
    store := es.NewInMemoryEventStore()
    repo := es.NewRepository(store)
    
    // Create a new bank account
    aggregateID := uuid.New()
    account := bankaccount.NewBankAccount(aggregateID, "ACC-001")
    
    // Perform business operations
    if err := account.Open(1000); err != nil {
        log.Fatal(err)
    }
    
    if err := account.Deposit(500); err != nil {
        log.Fatal(err)
    }
    
    if err := account.Withdraw(200); err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Balance: %d\n", account.GetBalance()) // Balance: 1300
    
    // Save the aggregate
    if err := repo.Save(context.Background(), account); err != nil {
        log.Fatal(err)
    }
    
    // Later, load the aggregate
    loadedAccount := bankaccount.NewBankAccount(aggregateID, "ACC-001")
    
    if err := repo.Load(context.Background(), loadedAccount); err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Loaded balance: %d\n", loadedAccount.GetBalance()) // Loaded balance: 1300
}
```

## Step 4: Add Tests

Write tests to verify your business logic:

```go
package bankaccount

import (
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
)

func TestShouldOpenAccountWithInitialBalance(t *testing.T) {
    // Arrange
    account := NewBankAccount(uuid.New(), "ACC-001")
    
    // Act
    err := account.Open(1000)
    
    // Assert
    assert.NoError(t, err)
    assert.True(t, account.IsOpen())
    assert.Equal(t, int64(1000), account.GetBalance())
}

func TestShouldDepositMoneyWhenAccountIsOpen(t *testing.T) {
    // Arrange
    account := NewBankAccount(uuid.New(), "ACC-001")
    err := account.Open(1000)
    assert.NoError(t, err)
    
    // Act
    err = account.Deposit(500)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, int64(1500), account.GetBalance())
}

func TestShouldReturnErrorWhenWithdrawingMoreThanBalance(t *testing.T) {
    // Arrange
    account := NewBankAccount(uuid.New(), "ACC-001")
    err := account.Open(1000)
    assert.NoError(t, err)
    
    // Act
    err = account.Withdraw(1500)
    
    // Assert
    assert.Error(t, err)
    assert.Equal(t, int64(1000), account.GetBalance()) // Balance unchanged
}
```

## Next Steps

- Explore advanced features like multi-tenancy
- Implement custom event stores for persistence
- Add event publishing for integration with other systems
- Consider event versioning strategies for schema evolution
- Learn about projections and read models for queries