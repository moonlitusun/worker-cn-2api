package store

import "testing"

func TestTraeCredentialsPersistenceAndRollback(t *testing.T) {
	s := tempStore(t)
	if err := s.SetTraeCredentials("token", "user"); err != nil {
		t.Fatal(err)
	}
	reloaded, err := New(s.filePath)
	if err != nil {
		t.Fatal(err)
	}
	if token, user := reloaded.GetTraeCredentials(); token != "token" || user != "user" {
		t.Fatal("credentials did not round-trip")
	}
	s.filePath = t.TempDir() + "/missing/data.json"
	if err := s.SetTraeCredentials("new", "other"); err == nil {
		t.Fatal("expected write failure")
	}
	if token, user := s.GetTraeCredentials(); token != "token" || user != "user" {
		t.Fatal("failed save changed credentials")
	}
}
