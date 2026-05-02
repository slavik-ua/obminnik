package domain

import (
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

	// The Buyer is buying baseAsset (BTC) and must pay with the quote asset (USD). The engine checks if they have enough Locked USD to cover the price
	buyerQuoteBalance, ok := b.balances[buyer][quoteAsset]
	if !ok || buyerQuoteBalance.Locked < quoteAmount {
		return ErrInsufficientFunds
	}

	// The seller sells baseAsset (BTC). The engine is checking if they have enough Locked BTC
	sellerBaseBalance, ok := b.balances[seller][baseAsset]
	if !ok || sellerBaseBalance.Locked < size {
		return ErrInsufficientFunds
	}

	buyerQuoteBalance.Locked -= quoteAmount
	b.Deposit(buyer, baseAsset, size)

	sellerBaseBalance.Locked -= size
	b.Deposit(seller, quoteAsset, quoteAmount)

	return nil
}
