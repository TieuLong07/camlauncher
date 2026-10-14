package auth

import (
	"Aether/pkg/models"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GenerateOfflineAccount creates a crack/offline account with standard Mojang-compliant UUID v3
func GenerateOfflineAccount(username string) (*models.Account, error) {
	if len(username) < 3 || len(username) > 16 {
		return nil, fmt.Errorf("tên nhân vật phải từ 3 đến 16 ký tự")
	}

	// Mojang UUID v3 specification: UUID.nameUUIDFromBytes(("OfflinePlayer:" + username).getBytes(StandardCharsets.UTF_8))
	hasher := md5.New()
	hasher.Write([]byte("OfflinePlayer:" + username))
	hash := hasher.Sum(nil)

	// Set version to 3 (0011) and variant to RFC 4122 (10xx)
	hash[6] = (hash[6] & 0x0f) | 0x30
	hash[8] = (hash[8] & 0x3f) | 0x80

	hexStr := hex.EncodeToString(hash)
	formattedUUID := fmt.Sprintf("%s-%s-%s-%s-%s",
		hexStr[0:8],
		hexStr[8:12],
		hexStr[12:16],
		hexStr[16:20],
		hexStr[20:32],
	)

	// Skin & Avatar URLs (Minotar renders head/helm in 2.5D or 2D)
	skinURL := fmt.Sprintf("https://minotar.net/skin/%s", username)

	account := &models.Account{
		ID:          uuid.New().String(),
		Username:    username,
		Type:        models.AccountOffline,
		UUID:        formattedUUID,
		AccessToken: formattedUUID, // In offline mode, token is arbitrary or uuid
		SkinURL:     skinURL,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	return account, nil
}
