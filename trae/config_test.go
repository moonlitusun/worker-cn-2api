package trae

import "testing"

func TestDesktopImportRequiresOptIn(t *testing.T) {
	t.Setenv("WORKER_TRAE_TOKEN", "")
	t.Setenv("WORKER_TRAE_USER_ID", "")
	for _, flag := range []string{"", "false", "invalid", "true", "1"} {
		t.Run(flag, func(t *testing.T) {
			t.Setenv("WORKER_IMPORT_DESKTOP_CREDENTIALS", flag)
			called := false
			cfg := resolveConfig("", "", func() (Config, error) { called = true; return Config{Token: "synthetic-desktop-token"}, nil })
			want := flag == "true" || flag == "1"
			if called != want || (cfg.Token != "") != want {
				t.Fatal("unexpected desktop import", flag, called)
			}
		})
	}
	t.Setenv("WORKER_IMPORT_DESKTOP_CREDENTIALS", "true")
	cfg := resolveConfig("explicit-token", "", func() (Config, error) {
		t.Fatal("must not read desktop when explicitly configured")
		return Config{}, nil
	})
	if cfg.Token != "explicit-token" {
		t.Fatal("explicit credential lost")
	}
}
