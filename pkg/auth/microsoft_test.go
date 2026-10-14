package auth

import (
	"context"
	"testing"
	"time"
)

func TestFormatUUID(t *testing.T) {
	raw := "e96fbb52f5423e5494427a4cb8c1e285"
	expected := "e96fbb52-f542-3e54-9442-7a4cb8c1e285"
	formatted := FormatUUID(raw)
	if formatted != expected {
		t.Errorf("Expected %s, got %s", expected, formatted)
	}

	alreadyFormatted := "e96fbb52-f542-3e54-9442-7a4cb8c1e285"
	if FormatUUID(alreadyFormatted) != expected {
		t.Errorf("Expected %s, got %s", expected, FormatUUID(alreadyFormatted))
	}
}

func TestRequestDeviceCode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	res, err := RequestDeviceCode(ctx)
	if err != nil {
		t.Fatalf("RequestDeviceCode failed: %v", err)
	}

	if res.UserCode == "" {
		t.Errorf("Expected non-empty UserCode")
	}
	if res.DeviceCode == "" {
		t.Errorf("Expected non-empty DeviceCode")
	}
	if res.VerificationURI == "" {
		t.Errorf("Expected non-empty VerificationURI")
	}
	if res.Interval <= 0 {
		t.Errorf("Expected Interval > 0")
	}
	t.Logf("Got DeviceCode: %s, UserCode: %s, URI: %s, Interval: %d", res.DeviceCode[:10]+"...", res.UserCode, res.VerificationURI, res.Interval)
}
