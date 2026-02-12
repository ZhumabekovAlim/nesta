package robokassa

import "testing"

func TestEqualAmountByCurrency(t *testing.T) {
	a, _ := ParseAmount("100.00")
	b, _ := ParseAmount("100.000000")
	if !EqualAmountByCurrency(a, b, "KZT") {
		t.Fatal("amounts must be equal after quantization")
	}
}
