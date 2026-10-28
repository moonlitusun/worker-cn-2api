// Package trae adapts the Trae CN solo_work_lite compatibility chat protocol.
package trae

import (
	"os"
	"strings"
)

const DefaultBaseURL = "https://trae-api-cn.mchost.guru"
const DefaultEnterpriseBaseURL = "https://api.enterprise.trae.cn"
const EnterpriseClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
const Function = "solo_work_lite"

type Config struct {
	EnterpriseURL                              string
	Token, UserID, DeviceID, MachineID         string
	BaseURL, Version, VersionCode              string
	RefreshToken, ExpiresAt, ClientID, AuthURL string
}

func ResolveConfig(token, userID string) Config {
	if token == "" {
		token = os.Getenv("WORKER_TRAE_TOKEN")
	}
	if userID == "" {
		userID = os.Getenv("WORKER_TRAE_USER_ID")
	}
	c := Config{EnterpriseURL: os.Getenv("WORKER_TRAE_ENTERPRISE_URL"), Token: strings.TrimSpace(token), UserID: strings.TrimSpace(userID),
		DeviceID: os.Getenv("WORKER_TRAE_DEVICE_ID"), MachineID: os.Getenv("WORKER_TRAE_MACHINE_ID"),
		BaseURL: os.Getenv("WORKER_TRAE_BASE_URL"), Version: os.Getenv("WORKER_TRAE_VERSION"), VersionCode: os.Getenv("WORKER_TRAE_VERSION_CODE"),
		RefreshToken: os.Getenv("WORKER_TRAE_REFRESH_TOKEN"), ExpiresAt: os.Getenv("WORKER_TRAE_EXPIRES_AT"), ClientID: os.Getenv("WORKER_TRAE_CLIENT_ID"), AuthURL: os.Getenv("WORKER_TRAE_AUTH_URL")}
	if c.Token == "" {
		if local, err := detectDesktop(); err == nil {
			c.Token = local.Token
			c.UserID = local.UserID
			c.RefreshToken = local.RefreshToken
			c.ExpiresAt = local.ExpiresAt
			c.ClientID = local.ClientID
		}
	}
	return defaults(c)
}

func defaults(c Config) Config {
	if c.EnterpriseURL == "" {
		c.EnterpriseURL = DefaultEnterpriseBaseURL
	}
	c.EnterpriseURL = strings.TrimRight(c.EnterpriseURL, "/")
	if c.ClientID == "" {
		c.ClientID = "ono9krqynydwx5"
	}
	if c.AuthURL == "" {
		c.AuthURL = "https://api.trae.com.cn"
	}
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	// A documented compatibility profile, not the host's VS Code version.
	if c.Version == "" {
		c.Version = "0.1.61"
	}
	if c.VersionCode == "" {
		c.VersionCode = "20260820"
	}
	return c
}
