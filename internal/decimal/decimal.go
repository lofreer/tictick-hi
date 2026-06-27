package decimal

import (
	"fmt"
	"math/big"
)

type Decimal struct {
	value *big.Rat
}

func Zero() Decimal {
	return Decimal{value: new(big.Rat)}
}

func NewInt(value int64) Decimal {
	return Decimal{value: big.NewRat(value, 1)}
}

func Parse(value string) (Decimal, error) {
	parsed, ok := new(big.Rat).SetString(value)
	if !ok {
		return Decimal{}, fmt.Errorf("invalid decimal %q", value)
	}
	return Decimal{value: parsed}, nil
}

func (decimal Decimal) Add(other Decimal) Decimal {
	return Decimal{value: new(big.Rat).Add(decimal.value, other.value)}
}

func (decimal Decimal) Sub(other Decimal) Decimal {
	return Decimal{value: new(big.Rat).Sub(decimal.value, other.value)}
}

func (decimal Decimal) Mul(other Decimal) Decimal {
	return Decimal{value: new(big.Rat).Mul(decimal.value, other.value)}
}

func (decimal Decimal) Quo(other Decimal) Decimal {
	return Decimal{value: new(big.Rat).Quo(decimal.value, other.value)}
}

func (decimal Decimal) Positive() bool {
	return decimal.value.Sign() > 0
}

func (decimal Decimal) GreaterThan(other Decimal) bool {
	return decimal.value.Cmp(other.value) > 0
}

func (decimal Decimal) String() string {
	return decimal.Format(8)
}

func (decimal Decimal) Format(precision int) string {
	return decimal.value.FloatString(precision)
}
