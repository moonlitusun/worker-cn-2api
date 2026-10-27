package store

import "testing"

func TestNamedTraeAccountsRefreshSwitchAndRollback(t *testing.T) {
	s := tempStore(t)
	if _, err := s.ChangeTraeAccount("save", "", "primary", TraeSession{Token: "token-a", RefreshToken: "refresh-a"}); err != nil {
		t.Fatal(err)
	}
	accounts, active := s.ListTraeAccounts()
	first := active
	if len(accounts) != 1 || first == "" {
		t.Fatal("account not created")
	}
	if err := s.SetTraeSession(TraeSession{Token: "rotated-a", RefreshToken: "rotated-refresh"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ChangeTraeAccount("save", "", "backup", TraeSession{Token: "token-b"}); err != nil {
		t.Fatal(err)
	}
	accounts, second := s.ListTraeAccounts()
	if second == first || len(accounts) != 2 {
		t.Fatal("account was overwritten")
	}
	session, err := s.ChangeTraeAccount("activate", first, "", TraeSession{})
	if err != nil || session.Token != "rotated-a" {
		t.Fatal("switch restored stale token", err)
	}
	reloaded, err := New(s.filePath)
	if err != nil {
		t.Fatal(err)
	}
	_, restored := reloaded.ListTraeAccounts()
	if restored != first {
		t.Fatal("active selection not persisted")
	}
	if _, err := s.ChangeTraeAccount("delete", second, "", TraeSession{}); err != nil {
		t.Fatal(err)
	}
	if token, _ := s.GetTraeCredentials(); token != "rotated-a" {
		t.Fatal("deleting inactive account changed active credentials")
	}
	s.filePath = t.TempDir() + "/missing/data.json"
	if _, err := s.ChangeTraeAccount("delete", first, "", TraeSession{}); err == nil {
		t.Fatal("expected disk failure")
	}
	accounts, active = s.ListTraeAccounts()
	if len(accounts) != 1 || active != first {
		t.Fatal("failed save mutated profiles")
	}
}
