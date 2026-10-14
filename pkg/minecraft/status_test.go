package minecraft

import (
	"testing"
)

func TestGetServerLiveStatus(t *testing.T) {
	status := GetServerLiveStatus()
	if status == nil {
		t.Fatalf("Expected non-nil status")
	}
	t.Logf("Server Status: Online=%v, Ping=%d ms, Players=%d/%d, Version=%s, MOTD=%s",
		status.Online, status.PingMs, status.Players, status.MaxPlayers, status.Version, status.MOTD)
}
