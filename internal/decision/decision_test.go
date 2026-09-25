package decision

import (
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"testing"
)

const fixedOpportunity = "20000000-0000-4000-8000-000000000001"

func TestNormalize(t *testing.T) {
	value, err := Normalize(Opportunity{ID: fixedOpportunity, Placement: " Search_Results ", Country: " gb "})
	if err != nil {
		t.Fatalf("Normalize(): %v", err)
	}
	if value.Placement != "search_results" || value.Country != "GB" || value.ID != fixedOpportunity {
		t.Fatalf("unexpected normalization: %+v", value)
	}

	invalid := []Opportunity{
		{ID: "00000000-0000-0000-0000-000000000000", Placement: "search_results", Country: "GB"},
		{ID: "20000000-0000-4000-8000-00000000000A", Placement: "search_results", Country: "GB"},
		{ID: fixedOpportunity, Placement: "1invalid", Country: "GB"},
		{ID: fixedOpportunity, Placement: "search_results", Country: "G1"},
	}
	for _, input := range invalid {
		if _, err := Normalize(input); err == nil {
			t.Fatalf("Normalize(%+v) succeeded", input)
		}
	}
}

func TestKnownRankingDigest(t *testing.T) {
	digest := score(fixedOpportunity, fixedCampaigns()[0])
	if got, want := hex.EncodeToString(digest[:]), "a2ffa7b136334a48e80276a806578157"; got != want {
		t.Fatalf("digest=%s want=%s", got, want)
	}
}

func TestProductionSelectionDoesNotReadBudget(t *testing.T) {
	if strings.Contains(strings.ToLower(selectionSQL), "budget") {
		t.Fatalf("production selection query contains a budget concept: %s", selectionSQL)
	}
}

func TestRankingProperties(t *testing.T) {
	values := candidates(fixedCampaigns())
	winner, ok := choose(fixedOpportunity, values)
	if !ok {
		t.Fatal("candidate set unexpectedly empty")
	}

	reversed := slices.Clone(values)
	slices.Reverse(reversed)
	reorderedWinner, _ := choose(fixedOpportunity, reversed)
	if reorderedWinner.ID != winner.ID {
		t.Fatalf("winner changed with input order: %s != %s", winner.ID, reorderedWinner.ID)
	}

	loserID := values[0].ID
	if loserID == winner.ID {
		loserID = values[1].ID
	}
	withoutLoser := slices.DeleteFunc(slices.Clone(values), func(value candidate) bool { return value.ID == loserID })
	if next, _ := choose(fixedOpportunity, withoutLoser); next.ID != winner.ID {
		t.Fatalf("removing non-winner changed winner: %s != %s", next.ID, winner.ID)
	}

	ranked := slices.Clone(values)
	for index := range ranked {
		ranked[index].Score = score(fixedOpportunity, ranked[index].ID)
	}
	slices.SortFunc(ranked, func(left, right candidate) int {
		if better(left, right) {
			return -1
		}
		if better(right, left) {
			return 1
		}
		return 0
	})
	withoutWinner := slices.DeleteFunc(slices.Clone(values), func(value candidate) bool { return value.ID == winner.ID })
	if next, _ := choose(fixedOpportunity, withoutWinner); next.ID != ranked[1].ID {
		t.Fatalf("next winner=%s want=%s", next.ID, ranked[1].ID)
	}
}

func TestRankingTieUsesLowestCampaignUUID(t *testing.T) {
	left := candidate{ID: fixedCampaigns()[0], Score: [16]byte{42}}
	right := candidate{ID: fixedCampaigns()[1], Score: [16]byte{42}}
	if !better(left, right) || better(right, left) {
		t.Fatal("equal digest did not use campaign UUID ascending")
	}
}

func TestGoldenRankingCorpus(t *testing.T) {
	want := []int{102, 113, 90, 109, 118, 87, 94, 103, 87, 97}
	ids := fixedCampaigns()
	counts := make(map[string]int, len(ids))
	for index := 1; index <= 1000; index++ {
		opportunityID := fmt.Sprintf("20000000-0000-4000-8000-%012x", index)
		winner, ok := choose(opportunityID, candidates(ids))
		if !ok {
			t.Fatal("empty candidate set")
		}
		counts[winner.ID]++
	}
	for index, id := range ids {
		if counts[id] != want[index] {
			t.Fatalf("winner count for %s=%d want=%d", id, counts[id], want[index])
		}
		if counts[id] == 0 {
			t.Fatalf("campaign %s never won", id)
		}
	}
}

func TestRejectionReasonOrder(t *testing.T) {
	value := candidate{State: "PAUSED", Placement: "home_feed", CountryTargeted: false}
	reasons := rejectionReasons(value, Opportunity{Placement: "search_results"})
	want := []string{"campaign_not_active", "placement_mismatch", "country_not_targeted"}
	if !slices.Equal(reasons, want) {
		t.Fatalf("reasons=%v want=%v", reasons, want)
	}
	eligible := rejectionReasons(candidate{State: "ACTIVE", Placement: "search_results", CountryTargeted: true}, Opportunity{Placement: "search_results"})
	if eligible == nil || len(eligible) != 0 {
		t.Fatalf("eligible reasons must be a non-nil empty array: %#v", eligible)
	}
}

func TestRandomUUIDIsCanonicalVersionFourAndUnique(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for range 100 {
		value, err := randomUUID()
		if err != nil {
			t.Fatal(err)
		}
		if !lowerUUIDPattern.MatchString(value) || value[14] != '4' || value[19] != '8' && value[19] != '9' && value[19] != 'a' && value[19] != 'b' {
			t.Fatalf("invalid UUIDv4: %s", value)
		}
		if _, duplicate := seen[value]; duplicate {
			t.Fatalf("duplicate UUID: %s", value)
		}
		seen[value] = struct{}{}
	}
}

func fixedCampaigns() []string {
	values := make([]string, 10)
	for index := range values {
		values[index] = fmt.Sprintf("10000000-0000-4000-8000-%012x", index+1)
	}
	return values
}

func candidates(ids []string) []candidate {
	values := make([]candidate, len(ids))
	for index, id := range ids {
		values[index].ID = id
	}
	return values
}
