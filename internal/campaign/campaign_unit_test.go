package campaign

import (
	"reflect"
	"testing"
)

func TestNormalizeCreateProducesCanonicalSemanticCommand(t *testing.T) {
	command, err := normalizeCreate(CreateCommand{
		AdvertiserID: "00000000-0000-4000-8000-000000000001",
		Name:         "  Lunch  ", PlacementCode: " HOME_FEED ", AmountMinor: 1250,
		Currency: " eur ", Countries: []string{"fr", " DE ", "FR"}, IdempotencyKey: "key",
	})
	if err != nil {
		t.Fatalf("normalizeCreate() unexpected error: %v", err)
	}
	if command.Name != "Lunch" || command.PlacementCode != "home_feed" || command.Currency != "EUR" {
		t.Fatalf("normalizeCreate() did not normalize scalar fields: %+v", command)
	}
	if !reflect.DeepEqual(command.Countries, []string{"DE", "FR"}) {
		t.Fatalf("countries = %v, want [DE FR]", command.Countries)
	}
}

func TestNormalizeCreateRejectsInvalidConfiguration(t *testing.T) {
	_, err := normalizeCreate(CreateCommand{IdempotencyKey: "key"})
	if err == nil {
		t.Fatal("normalizeCreate() accepted incomplete configuration")
	}
	for _, field := range []string{"name", "placement_code", "configured_amount_minor", "currency", "countries"} {
		if _, exists := err.Fields[field]; !exists {
			t.Fatalf("validation error missing %q: %v", field, err.Fields)
		}
	}
}
