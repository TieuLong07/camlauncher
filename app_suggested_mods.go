package main

import (
	"Aether/pkg/models"
	modsync "Aether/pkg/sync"
	"fmt"
)

func (a *App) GetSuggestedMods(serverID string) ([]models.SuggestedMod, error) {
	instance := a.store.FindInstanceByServerID(serverID)
	if instance == nil {
		return modsync.SuggestedMods(serverID), nil
	}
	return modsync.InspectSuggestedMods(instance.Path, serverID)
}

func (a *App) SetSuggestedModInstalled(serverID, modID string, install bool) error {
	a.cmdMu.Lock()
	defer a.cmdMu.Unlock()
	if a.isRunning || a.isStarting {
		return fmt.Errorf("hãy đóng Minecraft trước khi thay đổi mod")
	}
	instance := a.store.FindInstanceByServerID(serverID)
	if instance == nil {
		return fmt.Errorf("hãy cài modpack trước khi cài mod tùy chỉnh")
	}
	if instance.GameVersion != "1.20.1" || instance.Loader != "forge" {
		return fmt.Errorf("mod gợi ý này cần Forge 1.20.1")
	}
	return modsync.SetSuggestedMod(instance.Path, serverID, modID, install)
}
