package storage

import (
	"Aether/pkg/models"
	"testing"
)

func TestFindOfficialChillTownPrefersServerIDThenLegacyCookingName(t *testing.T) {
	instances := []models.Instance{
		{ID: "boosted", Name: "TiCa Boosted 1.21.1", GameVersion: "1.21.1", Loader: "fabric"},
		{ID: "legacy", Name: "Cooking TQ", GameVersion: "1.20.1", Loader: "forge"},
		{ID: "official", Name: "Chill Town", ServerID: "chilltown", GameVersion: "1.20.1", Loader: "forge"},
	}
	got := FindOfficialChillTown(instances)
	if got == nil || got.ID != "official" {
		t.Fatalf("expected official Chill Town, got %#v", got)
	}
}

func TestFindOfficialChillTownAcceptsLegacyCookingProfile(t *testing.T) {
	got := FindOfficialChillTown([]models.Instance{{ID: "legacy", Name: "Cooking TQ"}})
	if got == nil || got.ID != "legacy" {
		t.Fatalf("expected legacy Cooking TQ, got %#v", got)
	}
}
