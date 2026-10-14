package auth

import (
	"Aether/pkg/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MicrosoftClientID   = "9c1f1f43-58d5-4b7a-af0d-4e487f073441"
	DeviceCodeURL       = "https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode"
	TokenURL            = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
	XboxLiveAuthURL     = "https://user.auth.xboxlive.com/user/authenticate"
	XSTSAuthURL         = "https://xsts.auth.xboxlive.com/xsts/authorize"
	MinecraftAuthURL    = "https://api.minecraftservices.com/authentication/login_with_xbox"
	MinecraftProfileURL = "https://api.minecraftservices.com/minecraft/profile"
)

// DeviceCodeResult holds the response from Microsoft OAuth device code endpoint
type DeviceCodeResult struct {
	UserCode        string `json:"userCode"`
	DeviceCode      string `json:"deviceCode"`
	VerificationURI string `json:"verificationUri"`
	ExpiresIn       int    `json:"expiresIn"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

// RequestDeviceCode initiates Microsoft Device Authorization Grant
func RequestDeviceCode(ctx context.Context) (*DeviceCodeResult, error) {
	data := url.Values{
		"client_id": {MicrosoftClientID},
		"scope":     {"XboxLive.signin offline_access"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", DeviceCodeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("không thể tạo yêu cầu device code: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lỗi kết nối Microsoft: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lỗi đọc phản hồi từ Microsoft: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Microsoft trả về lỗi (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		UserCode        string `json:"user_code"`
		DeviceCode      string `json:"device_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
		Message         string `json:"message"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("lỗi parse dữ liệu device code: %w", err)
	}

	interval := res.Interval
	if interval < 2 {
		interval = 5
	}

	return &DeviceCodeResult{
		UserCode:        res.UserCode,
		DeviceCode:      res.DeviceCode,
		VerificationURI: res.VerificationURI,
		ExpiresIn:       res.ExpiresIn,
		Interval:        interval,
		Message:         res.Message,
	}, nil
}

type msTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// PollForMSToken polls Microsoft token endpoint until user approves or code expires
func PollForMSToken(ctx context.Context, deviceCode string, interval int, expiresIn int) (*msTokenResponse, error) {
	if interval < 2 {
		interval = 5
	}
	pollDuration := time.Duration(interval) * time.Second
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)

	client := &http.Client{Timeout: 15 * time.Second}

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("mã xác thực Microsoft đã hết hạn (quá thời gian chờ)")
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollDuration):
		}

		data := url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"client_id":   {MicrosoftClientID},
			"device_code": {deviceCode},
		}

		req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		if err != nil {
			// Temporary network hiccup, retry next interval
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var tokRes msTokenResponse
		if err := json.Unmarshal(body, &tokRes); err != nil {
			continue
		}

		if resp.StatusCode == http.StatusOK && tokRes.AccessToken != "" {
			return &tokRes, nil
		}

		switch tokRes.Error {
		case "authorization_pending":
			// User has not logged in yet, continue polling
			continue
		case "slow_down":
			// Server requests slower polling rate
			pollDuration += 5 * time.Second
			continue
		case "expired_token":
			return nil, fmt.Errorf("phiên xác thực đã hết hạn, vui lòng thao tác lại")
		case "access_denied":
			return nil, fmt.Errorf("bạn đã từ chối cấp quyền trên Microsoft")
		default:
			if tokRes.Error != "" {
				desc := tokRes.ErrorDesc
				if desc == "" {
					desc = tokRes.Error
				}
				return nil, fmt.Errorf("lỗi xác thực Microsoft: %s", desc)
			}
		}
	}
}

// RefreshMSToken exchanges a refresh token for new access and refresh tokens
func RefreshMSToken(ctx context.Context, refreshToken string) (*msTokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {MicrosoftClientID},
		"refresh_token": {refreshToken},
		"scope":         {"XboxLive.signin offline_access"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lỗi kết nối làm mới token Microsoft: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokRes msTokenResponse
	if err := json.Unmarshal(body, &tokRes); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK || tokRes.AccessToken == "" {
		desc := tokRes.ErrorDesc
		if desc == "" {
			desc = tokRes.Error
		}
		return nil, fmt.Errorf("không thể làm mới phiên đăng nhập Microsoft (%s)", desc)
	}

	return &tokRes, nil
}

// AuthenticateXboxLive exchanges Microsoft access token for Xbox Live User Token and user hash
func AuthenticateXboxLive(ctx context.Context, msAccessToken string) (token string, userHash string, err error) {
	payload := map[string]interface{}{
		"Properties": map[string]interface{}{
			"AuthMethod": "RPS",
			"SiteName":   "user.auth.xboxlive.com",
			"RpsTicket":  "d=" + msAccessToken,
		},
		"RelyingParty": "http://auth.xboxlive.com",
		"TokenType":    "JWT",
	}

	jsonBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", XboxLiveAuthURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("lỗi kết nối Xbox Live: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("Xbox Live từ chối xác thực (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		Token         string `json:"Token"`
		DisplayClaims struct {
			Xui []struct {
				UHS string `json:"uhs"`
			} `json:"xui"`
		} `json:"DisplayClaims"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", "", err
	}

	if res.Token == "" || len(res.DisplayClaims.Xui) == 0 || res.DisplayClaims.Xui[0].UHS == "" {
		return "", "", fmt.Errorf("không nhận được mã xác thực Xbox Live hợp lệ")
	}

	return res.Token, res.DisplayClaims.Xui[0].UHS, nil
}

// AuthenticateXSTS exchanges Xbox Live token for XSTS token
func AuthenticateXSTS(ctx context.Context, xblToken string) (token string, err error) {
	payload := map[string]interface{}{
		"Properties": map[string]interface{}{
			"SandboxId":  "RETAIL",
			"UserTokens": []string{xblToken},
		},
		"RelyingParty": "rp://api.minecraftservices.com/",
		"TokenType":    "JWT",
	}

	jsonBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", XSTSAuthURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("lỗi kết nối máy chủ bảo mật XSTS: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var res struct {
		Token string `json:"Token"`
		XErr  uint64 `json:"XErr"`
	}
	_ = json.Unmarshal(body, &res)

	if resp.StatusCode != http.StatusOK {
		switch res.XErr {
		case 2148916233:
			return "", fmt.Errorf("tài khoản Microsoft chưa được liên kết với hồ sơ Xbox. Vui lòng vào xbox.com để đăng nhập tạo Gamertag trước")
		case 2148916235:
			return "", fmt.Errorf("dịch vụ Xbox Live hiện chưa hỗ trợ quốc gia hoặc vùng lãnh thổ của tài khoản này")
		case 2148916236, 2148916237:
			return "", fmt.Errorf("tài khoản Microsoft yêu cầu xác minh độ tuổi người lớn từ phụ huynh")
		case 2148916238:
			return "", fmt.Errorf("đây là tài khoản trẻ em (Child account), cần được tài khoản phụ huynh thêm vào nhóm gia đình để cấp quyền")
		default:
			return "", fmt.Errorf("lỗi cấp quyền XSTS (mã %d): %s", res.XErr, string(body))
		}
	}

	if res.Token == "" {
		return "", fmt.Errorf("không nhận được XSTS token hợp lệ")
	}

	return res.Token, nil
}

// AuthenticateMinecraft exchanges userHash and XSTS token for Minecraft Services JWT
func AuthenticateMinecraft(ctx context.Context, userHash string, xstsToken string) (accessToken string, expiresIn int, err error) {
	payload := map[string]interface{}{
		"identityToken": fmt.Sprintf("XBL3.0 x=%s;%s", userHash, xstsToken),
	}

	jsonBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", MinecraftAuthURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("lỗi kết nối dịch vụ Minecraft Services: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("dịch vụ Minecraft Services từ chối (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", 0, err
	}

	if res.AccessToken == "" {
		return "", 0, fmt.Errorf("không nhận được Minecraft Access Token")
	}

	return res.AccessToken, res.ExpiresIn, nil
}

type MinecraftProfile struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Skins []struct {
		ID      string `json:"id"`
		State   string `json:"state"`
		URL     string `json:"url"`
		Variant string `json:"variant"`
	} `json:"skins"`
}

// GetMinecraftProfile retrieves the player's username, UUID and active skin
func GetMinecraftProfile(ctx context.Context, mcAccessToken string) (*MinecraftProfile, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", MinecraftProfileURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+mcAccessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lỗi kiểm tra hồ sơ Minecraft: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("tài khoản Microsoft này chưa sở hữu bản quyền game Minecraft (chưa mua game hoặc gói Game Pass đã hết hạn)")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("không thể tải hồ sơ Minecraft (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var profile MinecraftProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("lỗi đọc dữ liệu hồ sơ: %w", err)
	}

	if profile.Name == "" || profile.ID == "" {
		return nil, fmt.Errorf("hồ sơ Minecraft không hợp lệ")
	}

	return &profile, nil
}

// FormatUUID converts a 32-character hex UUID to standard 8-4-4-4-12 hyphenated format
func FormatUUID(raw string) string {
	clean := strings.ReplaceAll(raw, "-", "")
	if len(clean) == 32 {
		return fmt.Sprintf("%s-%s-%s-%s-%s", clean[0:8], clean[8:12], clean[12:16], clean[16:20], clean[20:32])
	}
	return raw
}

// CompleteMicrosoftAuth executes full token exchange pipeline from Microsoft access & refresh tokens
func CompleteMicrosoftAuth(ctx context.Context, msAccessToken, msRefreshToken string) (*models.Account, error) {
	// 1. Xbox Live
	xblToken, userHash, err := AuthenticateXboxLive(ctx, msAccessToken)
	if err != nil {
		return nil, err
	}

	// 2. XSTS
	xstsToken, err := AuthenticateXSTS(ctx, xblToken)
	if err != nil {
		return nil, err
	}

	// 3. Minecraft Services
	mcToken, mcExpiresIn, err := AuthenticateMinecraft(ctx, userHash, xstsToken)
	if err != nil {
		return nil, err
	}

	// 4. Minecraft Profile
	profile, err := GetMinecraftProfile(ctx, mcToken)
	if err != nil {
		return nil, err
	}

	// Active skin or fallback Minotar
	skinURL := fmt.Sprintf("https://minotar.net/skin/%s", profile.Name)
	for _, s := range profile.Skins {
		if strings.EqualFold(s.State, "ACTIVE") && s.URL != "" {
			skinURL = s.URL
			break
		}
	}

	expiresAt := time.Now().Unix() + int64(mcExpiresIn)
	if mcExpiresIn <= 0 {
		expiresAt = time.Now().Unix() + 86400 // Default 24h
	}

	account := &models.Account{
		ID:           uuid.New().String(),
		Username:     profile.Name,
		Type:         models.AccountMicrosoft,
		UUID:         FormatUUID(profile.ID),
		AccessToken:  mcToken,
		RefreshToken: msRefreshToken,
		ExpiresAt:    expiresAt,
		SkinURL:      skinURL,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}

	return account, nil
}

// RefreshMicrosoftAccount refreshes an expired Microsoft account in-place
func RefreshMicrosoftAccount(ctx context.Context, acc *models.Account) error {
	if acc.Type != models.AccountMicrosoft || acc.RefreshToken == "" {
		return nil
	}

	tokRes, err := RefreshMSToken(ctx, acc.RefreshToken)
	if err != nil {
		return err
	}

	refreshed, err := CompleteMicrosoftAuth(ctx, tokRes.AccessToken, tokRes.RefreshToken)
	if err != nil {
		return err
	}

	acc.Username = refreshed.Username
	acc.UUID = refreshed.UUID
	acc.AccessToken = refreshed.AccessToken
	acc.RefreshToken = refreshed.RefreshToken
	acc.ExpiresAt = refreshed.ExpiresAt
	acc.SkinURL = refreshed.SkinURL
	return nil
}
