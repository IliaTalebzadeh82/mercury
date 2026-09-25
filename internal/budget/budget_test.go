package budget

import (
	"errors"
	"math"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestNormalizeAndFingerprintUseSemanticCommand(t *testing.T) {
	left, err := normalize(Command{
		CampaignID: " 11000000-0000-4000-8000-000000000001 ", AmountMinor: 25,
		Currency: " eur ", IdempotencyKey: "key-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	right, err := normalize(Command{
		CampaignID: "11000000-0000-4000-8000-000000000001", AmountMinor: 25,
		Currency: "EUR", IdempotencyKey: "different-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if left.CampaignID != right.CampaignID || left.Currency != right.Currency {
		t.Fatalf("normalization differs: %+v %+v", left, right)
	}
	if commandFingerprint(left) != commandFingerprint(right) {
		t.Fatal("raw idempotency key affected the semantic fingerprint")
	}
	right.AmountMinor++
	if commandFingerprint(left) == commandFingerprint(right) {
		t.Fatal("different amount produced the same fingerprint")
	}
}

func TestCommitErrorsAreClassifiedConservatively(t *testing.T) {
	if !errors.Is(classifyCommitError(pgx.ErrTxCommitRollback), ErrDatabase) {
		t.Fatal("known rollback was not classified as a database failure")
	}
	if !errors.Is(classifyCommitError(errors.New("connection lost during commit")), ErrOutcomeUnknown) {
		t.Fatal("ambiguous commit error was not classified as outcome unknown")
	}
}

func TestNormalizeRejectsInvalidFinancialInput(t *testing.T) {
	tests := []Command{
		{CampaignID: "bad", AmountMinor: 1, Currency: "EUR", IdempotencyKey: "key"},
		{CampaignID: "11000000-0000-4000-8000-000000000001", AmountMinor: 0, Currency: "EUR", IdempotencyKey: "key"},
		{CampaignID: "11000000-0000-4000-8000-000000000001", AmountMinor: 1, Currency: "JPY", IdempotencyKey: "key"},
		{CampaignID: "11000000-0000-4000-8000-000000000001", AmountMinor: 1, Currency: "EUR"},
	}
	for _, command := range tests {
		if _, err := normalize(command); err == nil {
			t.Fatalf("normalize(%+v) succeeded", command)
		}
	}
}

func TestMaxInt64AffordabilityArithmeticIsSafe(t *testing.T) {
	configured := int64(math.MaxInt64)
	committed := configured - 1
	remaining := configured - committed
	amount := int64(1)
	if amount > remaining {
		t.Fatal("exact final minor unit was rejected")
	}
	result := committed + amount
	if result != math.MaxInt64 {
		t.Fatalf("result=%d", result)
	}
	if amount = 2; amount <= remaining {
		t.Fatal("unaffordable amount passed subtraction comparison")
	}
}
