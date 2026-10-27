package workbuddy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	DefaultUpstreamURL = "https://www.workbuddy.cn/v2/chat/completions"
	DefaultClientVer   = "5.5.4"
	DefaultPlatform    = "darwin"
)

// DesktopAuthInfo matches the structure stored by the official WorkBuddy/CodeBuddy client.
type DesktopAuthInfo struct {
	UserID       string `json:"user_id"`
	AuthToken    string `json:"auth_token"`
	RefreshToken string `json:"refresh_token"`
	Region       string `json:"region"`
	ExpireAt     int64  `json:"expire_at"`
}

// Config holds the credentials and runtime parameters for WorkBuddy CN.
type Config struct {
	Token       string
	UserID      string
	Endpoint    string
	DeviceID    string
	ClientVer   string
	Platform    string
	Concurrency int
}

// ResolveConfig resolves the configuration using environment variables,
// explicit overrides, or falling back to local desktop client credentials.
func ResolveConfig(explicitToken string) *Config {
	token := strings.TrimSpace(explicitToken)
	if token == "" {
		token = os.Getenv("WORKER_WORKBUDDY_TOKEN")
	}
	if token == "" {
		token = os.Getenv("CODEBUDDY_AUTH_TOKEN")
	}

	userID := ""
	if token == "" {
		// Attempt to read from local desktop client config file
		if info, err := detectLocalDesktopAuth(); err == nil && info != nil {
			token = info.AuthToken
			userID = info.UserID
		}
	}

	endpoint := os.Getenv("WORKER_WORKBUDDY_ENDPOINT")
	if endpoint == "" {
		endpoint = DefaultUpstreamURL
	}

	clientVer := os.Getenv("WORKER_WORKBUDDY_CLIENT_VER")
	if clientVer == "" {
		clientVer = DefaultClientVer
	}

	plat := runtime.GOOS
	if plat == "" {
		plat = DefaultPlatform
	}

	deviceID := DeriveDeviceID(userID)

	return &Config{
		Token:       token,
		UserID:      userID,
		Endpoint:    endpoint,
		DeviceID:    deviceID,
		ClientVer:   clientVer,
		Platform:    plat,
		Concurrency: 2,
	}
}

// detectLocalDesktopAuth searches for the desktop client's authentication file on macOS / Windows.
func detectLocalDesktopAuth() (*DesktopAuthInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	var candidatePaths []string
	if runtime.GOOS == "darwin" {
		candidatePaths = append(candidatePaths, filepath.Join(home, "Library", "Application Support", "CodeBuddyExtension", "Data", "Public", "auth", "workbuddy-desktop.info"))
	} else if runtime.GOOS == "windows" {
		appData := os.Getenv("LOCALAPPDATA")
		if appData != "" {
			candidatePaths = append(candidatePaths, filepath.Join(appData, "CodeBuddyExtension", "Data", "Public", "auth", "workbuddy-desktop.info"))
		}
	} else {
		candidatePaths = append(candidatePaths, filepath.Join(home, ".local", "share", "CodeBuddyExtension", "Data", "Public", "auth", "workbuddy-desktop.info"))
	}

	for _, path := range candidatePaths {
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			var info DesktopAuthInfo
			var current struct {
				Auth struct {
					AccessToken string `json:"accessToken"`
					Domain      string `json:"domain"`
				}
				Account struct {
					UID string `json:"uid"`
				}
			}
			if json.Unmarshal(data, &current) == nil && current.Auth.AccessToken != "" && !strings.Contains(current.Auth.Domain, ".ai") {
				info.AuthToken = current.Auth.AccessToken
				info.UserID = current.Account.UID
			}
			if info.AuthToken == "" {
				_ = json.Unmarshal(data, &info)
			}
			if info.AuthToken != "" && info.Region != "intl" && info.Region != "international" {
				info.AuthToken = strings.TrimSpace(info.AuthToken)
				return &info, nil
			}
		}
	}
	return nil, os.ErrNotExist
}
