package domain

import (
	"fmt"
	"github.com/google/uuid"
	"math/big"
)

type Balance struct {
	Available int64
	Locked    int64
}

type BalanceCache struct {
	balances map[uuid.UUID]map[string]*Balance
}

func (b *BalanceCache) Clear() {
	b.balances = make(map[uuid.UUID]map[string]*Balance)
}

func NewBalanceCache() *BalanceCache {
	return &BalanceCache{
		balances: make(map[uuid.UUID]map[string]*Balance),
	}
}

func (b *BalanceCache) GetBalance(userID uuid.UUID, asset string) (*Balance, bool) {
	userBalances, ok := b.balances[userID]
	if !ok {
		return nil, false
	}
	balance, ok := userBalances[asset]
	return balance, ok
}

func (b *BalanceCache) InitBalance(userID uuid.UUID, asset string, available, locked int64) {
	if _, ok := b.balances[userID]; !ok {
		b.balances[userID] = make(map[string]*Balance)
	}

	b.balances[userID][asset] = &Balance{
		Available: available,
		Locked:    locked,
	}
}

func (b *BalanceCache) LockFunds(userID uuid.UUID, asset string, amount int64) error {
	balance, ok := b.balances[userID][asset]
	if !ok || balance.Available < amount {
		return ErrInsufficientFunds
	}

	balance.Available -= amount
	balance.Locked += amount
	return nil
}

func (b *BalanceCache) UnlockFunds(userID uuid.UUID, asset string, amount int64) error {
	balance, ok := b.balances[userID][asset]
	if !ok || balance.Locked < amount {
		return ErrInsufficientFunds
	}

	balance.Locked -= amount
	balance.Available += amount
	return nil
}

func (b *BalanceCache) Deposit(userID uuid.UUID, asset string, amount int64) {
	userBalances, ok := b.balances[userID]
	if !ok {
		userBalances = make(map[string]*Balance)
		b.balances[userID] = userBalances
	}

	balance, ok := userBalances[asset]
	if !ok {
		balance = &Balance{Available: 0, Locked: 0}
		userBalances[asset] = balance
	}

	balance.Available += amount
}

func (b *BalanceCache) SettleTrade(buyer, seller uuid.UUID, baseAsset, quoteAsset string, price, size int64) error {
	p := big.NewInt(price)
	s := big.NewInt(size)
	d := big.NewInt(Decimals)
	quoteAmountBig := new(big.Int).Mul(p, s)
	quoteAmountBig.Quo(quoteAmountBig, d)
	quoteAmount := quoteAmountBig.Int64()

	// The Buyer is buying baseAsset (BTC) and must pay with the quote asset (USD).
	userBids, ok := b.balances[buyer]
	if !ok {
		return fmt.Errorf("%w: buyer %s not found in cache", ErrInsufficientFunds, buyer)
	}
	buyerQuoteBalance, ok := userBids[quoteAsset]
	if !ok || buyerQuoteBalance.Locked < quoteAmount {
		locked := int64(0)
		if ok {
			locked = buyerQuoteBalance.Locked
		}
		return fmt.Errorf("%w: buyer %s insufficient %s locked (has %d, needs %d)", ErrInsufficientFunds, buyer, quoteAsset, locked, quoteAmount)
	}

	// The seller sells baseAsset (BTC).
	userAsks, ok := b.balances[seller]
	if !ok {
		return fmt.Errorf("%w: seller %s not found in cache", ErrInsufficientFunds, seller)
	}
	sellerBaseBalance, ok := userAsks[baseAsset]
	if !ok || sellerBaseBalance.Locked < size {
		locked := int64(0)
		if ok {
			locked = sellerBaseBalance.Locked
		}
		return fmt.Errorf("%w: seller %s insufficient %s locked (has %d, needs %d)", ErrInsufficientFunds, seller, baseAsset, locked, size)
	}

	buyerQuoteBalance.Locked -= quoteAmount
	b.Deposit(buyer, baseAsset, size)

	sellerBaseBalance.Locked -= size
	b.Deposit(seller, quoteAsset, quoteAmount)

	return nil
}
