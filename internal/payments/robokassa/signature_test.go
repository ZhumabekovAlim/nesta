package robokassa

import "testing"

func TestSignatureForInitMD5WithSortedShp(t *testing.T) {
	sig, err := SignatureForInit(HashMD5, "demo", "100.00", "123", "pass1", nil, map[string]string{
		"Shp_user": "u-1",
		"Shp_pay":  "p-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := hashSignature(HashMD5, "demo:100.00:123:pass1:Shp_pay=p-1:Shp_user=u-1")
	if sig != expected {
		t.Fatalf("unexpected signature: %s", sig)
	}
}

func TestSignatureForResultSHA256(t *testing.T) {
	sig, err := SignatureForResult(HashSHA256, "100.000000", "123", "pass2", map[string]string{"Shp_payment_id": "42"})
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := hashSignature(HashSHA256, "100.000000:123:pass2:Shp_payment_id=42")
	if sig != expected {
		t.Fatalf("unexpected signature: %s", sig)
	}
}

func TestConstantTimeEqualSignature(t *testing.T) {
	if !ConstantTimeEqualSignature("ABC", "abc") {
		t.Fatal("expected case insensitive equality")
	}
	if ConstantTimeEqualSignature("abc", "abd") {
		t.Fatal("expected mismatch")
	}
}
