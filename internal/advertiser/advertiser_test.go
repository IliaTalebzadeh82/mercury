package advertiser

import "testing"

func TestNormalizeName(t *testing.T) {
	name, err := normalizeName("  Mercury Kitchen  ")
	if err != nil || name != "Mercury Kitchen" {
		t.Fatalf("normalizeName() = %q, %v", name, err)
	}
	if _, err := normalizeName("   "); err == nil {
		t.Fatal("normalizeName() accepted an empty name")
	}
}

func TestValidateIdempotencyKeyBounds(t *testing.T) {
	if err := validateKey(""); err == nil {
		t.Fatal("validateKey() accepted an empty key")
	}
	if err := validateKey("phase-one"); err != nil {
		t.Fatalf("validateKey() rejected a valid key: %v", err)
	}
}
