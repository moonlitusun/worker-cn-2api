package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExistingQoderPATSurvivesOtherProviderUpdates(t *testing.T) {
	p := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(p, []byte(`{"pat":"existing-qoder-pat","workbuddy_token":"wb","password":"existing-password","api_keys":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := New(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetTraeCredentials("trae-token", "user"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddKey("test-key", ""); err != nil {
		t.Fatal(err)
	}
	reloaded, err := New(p)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.GetQoderPAT() != "existing-qoder-pat" || reloaded.GetWorkBuddyToken() != "wb" || reloaded.GetPassword() != "existing-password" {
		t.Fatal("existing channel credentials were lost")
	}
}
