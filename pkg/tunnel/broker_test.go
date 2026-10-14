package tunnel

import (
	"context"
	"testing"
	"time"
)

func TestRequestBrokerCredential(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	key, target, err := RequestBrokerCredential(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(key) < 20 || target == "" {
		t.Fatalf("invalid broker response: key=%t target=%q", key != "", target)
	}
}

func TestBrokerTunnelConnects(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	key, target, err := RequestBrokerCredential(ctx)
	if err != nil {
		t.Fatal(err)
	}
	manager := NewTunnelManager(nil)
	status, err := manager.Start(key, target)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Stop()
	if !status.Running || status.LocalPort == 0 {
		t.Fatalf("tunnel did not start: %#v", status)
	}
}
