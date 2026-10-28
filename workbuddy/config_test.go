package workbuddy

import "testing"

func TestDesktopImportRequiresOptIn(t *testing.T) {
	t.Setenv("WORKER_WORKBUDDY_TOKEN", "")
	t.Setenv("CODEBUDDY_AUTH_TOKEN", "")
	for _, flag := range []string{"", "false", "invalid", "true", "1"} {
		t.Run(flag, func(t *testing.T) {
			t.Setenv("WORKER_IMPORT_DESKTOP_CREDENTIALS", flag)
			called := false
			cfg := resolveConfig("", func() (*DesktopAuthInfo, error) {
				called = true
				return &DesktopAuthInfo{AuthToken: "synthetic-desktop-token"}, nil
			})
			want := flag == "true" || flag == "1"
			if called != want || (cfg.Token != "") != want {
				t.Fatal("unexpected desktop import", flag, called)
			}
		})
	}
	t.Setenv("WORKER_IMPORT_DESKTOP_CREDENTIALS", "true")
	cfg := resolveConfig("explicit-token", func() (*DesktopAuthInfo, error) {
		t.Fatal("must not read desktop when explicitly configured")
		return nil, nil
	})
	if cfg.Token != "explicit-token" {
		t.Fatal("explicit credential lost")
	}
}
