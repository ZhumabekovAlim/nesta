package robokassa

import (
	"fmt"
	"math/big"
	"strings"
)

var currencyScale = map[string]int{
	"KZT": 2,
	"RUB": 2,
	"USD": 2,
	"EUR": 2,
}

func ParseAmount(value string) (*big.Rat, error) {
	value = strings.TrimSpace(value)
	rat, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", value)
	}
	if rat.Sign() < 0 {
		return nil, fmt.Errorf("invalid negative amount: %s", value)
	}
	return rat, nil
}

func QuantizeAmount(value *big.Rat, currency string) *big.Rat {
	scale, ok := currencyScale[strings.ToUpper(currency)]
	if !ok {
		scale = 2
	}
	pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	num := new(big.Int).Mul(value.Num(), pow)
	den := value.Denom()
	q, r := new(big.Int).QuoRem(num, den, new(big.Int))
	doubleR := new(big.Int).Mul(r, big.NewInt(2))
	if doubleR.Cmp(den) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	return new(big.Rat).SetFrac(q, pow)
}

func EqualAmountByCurrency(expected, actual *big.Rat, currency string) bool {
	e := QuantizeAmount(expected, currency)
	a := QuantizeAmount(actual, currency)
	return e.Cmp(a) == 0
}

func FormatAmount(value *big.Rat, currency string) string {
	scale, ok := currencyScale[strings.ToUpper(currency)]
	if !ok {
		scale = 2
	}
	q := QuantizeAmount(value, currency)
	return q.FloatString(scale)
}
