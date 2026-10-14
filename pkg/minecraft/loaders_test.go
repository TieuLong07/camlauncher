package minecraft

import (
	"testing"
)

func TestGetLoaderVersionsFabric(t *testing.T) {
	res, err := GetLoaderVersions("1.21.1", "fabric")
	if err != nil {
		t.Fatalf("GetLoaderVersions failed: %v", err)
	}
	if len(res.Versions) == 0 {
		t.Errorf("Expected at least one fabric version for 1.21.1")
	}
	if res.Recommended == "" {
		t.Errorf("Expected recommended fabric version to be set")
	}
	t.Logf("Fabric 1.21.1: %d versions, recommended: %s", len(res.Versions), res.Recommended)
}

func TestGetLoaderVersionsForge(t *testing.T) {
	res, err := GetLoaderVersions("1.20.1", "forge")
	if err != nil {
		t.Fatalf("GetLoaderVersions failed: %v", err)
	}
	if len(res.Versions) == 0 {
		t.Errorf("Expected at least one forge version for 1.20.1")
	}
	if res.Recommended == "" {
		t.Errorf("Expected recommended forge version to be set")
	}
	t.Logf("Forge 1.20.1: %d versions, recommended: %s", len(res.Versions), res.Recommended)
}
