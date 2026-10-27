package trae

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDesktopCodecIntegrity(t *testing.T) {
	plain := []byte(`{"token":"synthetic-token","userId":"test"}`)
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i)
	}
	salt := make([]byte, 64)
	for i := range salt {
		salt[i] = codecSaltA[i] ^ codecSaltB[i]
	}
	hash := sha512.Sum512(seed)
	material := sha512.Sum512(append(hash[:], salt...))
	hash = sha512.Sum512(plain)
	payload := append(hash[:], plain...)
	pad := 16 - len(payload)%16
	payload = append(payload, []byte(strings.Repeat(string(byte(pad)), pad))...)
	block, _ := aes.NewCipher(material[:16])
	encrypted := make([]byte, len(payload))
	cipher.NewCBCEncrypter(block, material[16:32]).CryptBlocks(encrypted, payload)
	encoded := append([]byte("tc\x05\x10\x00\x00"), seed...)
	encoded = append(encoded, encrypted...)
	decoded, err := decodeDesktop(base64.StdEncoding.EncodeToString(encoded))
	if err != nil || string(decoded) != string(plain) {
		t.Fatalf("decode failed: %v", err)
	}
	encoded[len(encoded)-20] ^= 1
	if _, err = decodeDesktop(base64.StdEncoding.EncodeToString(encoded)); err == nil {
		t.Fatal("corrupt credentials accepted")
	}
	if _, err = decodeDesktop("not-base64"); err == nil {
		t.Fatal("malformed credentials accepted")
	}
}

func TestRefreshOnceAndPersistRotatedSession(t *testing.T) {
	var refreshes atomic.Int32
	var saves atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cloudide/api/v3/trae/oauth/ExchangeToken" {
			t.Error(r.URL.Path)
		}
		refreshes.Add(1)
		fmt.Fprintf(w, `{"Result":{"Token":"new-token","RefreshToken":"new-refresh","TokenExpireAt":%d}}`, time.Now().Add(time.Hour).UnixMilli())
	}))
	defer srv.Close()
	b := NewBridge(Config{Token: "old-token", RefreshToken: "old-refresh", ExpiresAt: time.Now().Add(-time.Hour).Format(time.RFC3339), AuthURL: srv.URL})
	b.SetCredentialSaver(func(c Config) error {
		if c.Token != "new-token" || c.RefreshToken != "new-refresh" {
			t.Error("wrong rotated session")
		}
		saves.Add(1)
		return nil
	})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := b.ensureCredentials(context.Background())
			if err != nil || c.Token != "new-token" {
				t.Errorf("refresh failed: %v", err)
			}
		}()
	}
	wg.Wait()
	if refreshes.Load() != 1 || saves.Load() != 1 {
		t.Fatal(refreshes.Load(), saves.Load())
	}
}

func TestAdminUpdateWinsOverInflightRefresh(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		fmt.Fprintf(w, `{"Result":{"Token":"stale-token","TokenExpireAt":%d}}`, time.Now().Add(time.Hour).UnixMilli())
	}))
	defer srv.Close()
	b := NewBridge(Config{Token: "old-token", RefreshToken: "old-refresh", ExpiresAt: time.Now().Add(-time.Hour).Format(time.RFC3339), AuthURL: srv.URL})
	b.SetCredentialSaver(func(Config) error { t.Error("stale session must not be saved"); return nil })
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, err := b.ensureCredentials(context.Background())
		if err != nil {
			t.Error(err)
		}
	}()
	<-started
	if err := b.ReplaceSession("admin-token", "admin-user", "", "", "", func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	close(release)
	<-done
	if b.Config().Token != "admin-token" || b.Config().RefreshToken != "" {
		t.Fatal("refresh overwrote admin credentials")
	}
}

func TestRotatedSessionSaveIsRetriedWithoutRefreshingAgain(t *testing.T) {
	var calls, saves atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprintf(w, `{"Result":{"Token":"new-token","RefreshToken":"new-refresh","TokenExpireAt":%d}}`, time.Now().Add(time.Hour).UnixMilli())
	}))
	defer srv.Close()
	b := NewBridge(Config{Token: "old-token", RefreshToken: "old-refresh", ExpiresAt: time.Now().Add(-time.Hour).Format(time.RFC3339), AuthURL: srv.URL})
	b.SetCredentialSaver(func(Config) error {
		if saves.Add(1) == 1 {
			return errors.New("disk unavailable")
		}
		return nil
	})
	if _, err := b.ensureCredentials(context.Background()); err == nil {
		t.Fatal("save failure was hidden")
	}
	c, err := b.ensureCredentials(context.Background())
	if err != nil || c.Token != "new-token" || calls.Load() != 1 || saves.Load() != 2 {
		t.Fatalf("unexpected retry: %v calls=%d saves=%d", err, calls.Load(), saves.Load())
	}
}

func TestCanceledRequestsDoNotWaitForRefreshOrCatalogGate(t *testing.T) {
	b := NewBridge(Config{Token: "token"})
	b.refreshGate <- struct{}{}
	b.catalogGate <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := b.ensureCredentials(ctx); err == nil {
			t.Error("refresh accepted cancellation")
		}
		if _, err := b.Models(ctx); err == nil {
			t.Error("catalog accepted cancellation")
		}
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("canceled request waited on gate")
	}
	<-b.refreshGate
	<-b.catalogGate
}

func TestRefreshFailureDoesNotChangeCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401); fmt.Fprint(w, "invalid old-refresh") }))
	defer srv.Close()
	b := NewBridge(Config{Token: "old-token", RefreshToken: "old-refresh", ExpiresAt: time.Now().Add(-time.Hour).Format(time.RFC3339), AuthURL: srv.URL})
	_, err := b.ensureCredentials(context.Background())
	if err == nil || strings.Contains(err.Error(), "old-refresh") || b.Config().Token != "old-token" {
		t.Fatal("refresh failure lost or exposed credentials")
	}
}

func TestSavingIdenticalSessionPreservesDiagnostics(t *testing.T) {
	b := NewBridge(Config{Token: "token"})
	o := b.observe()
	b.record("test", o, nil)
	if err := b.ChangeSession(func(c Config) (Config, error) { return c, nil }); err != nil {
		t.Fatal(err)
	}
	if b.Status().LastTest.At == "" || !b.Status().LastTest.Success {
		t.Fatal("profile-only change cleared connection diagnostics")
	}
}

func TestProfileChangeRetriesUnsavedRotationFirst(t *testing.T) {
	b := NewBridge(Config{Token: "rotated"})
	b.unsavedSession = true
	called := false
	b.SetCredentialSaver(func(c Config) error {
		if c.Token != "rotated" {
			t.Error("wrong session")
		}
		called = true
		return nil
	})
	if err := b.ChangeSession(func(c Config) (Config, error) {
		if !called {
			t.Fatal("account change preceded persistence of rotated credentials")
		}
		return c, nil
	}); err != nil {
		t.Fatal(err)
	}
	if b.Status().PersistencePending {
		t.Fatal("session still pending")
	}
}

func TestNamedAccountSwitchPreservesInflightRotation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	saved := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		fmt.Fprintf(w, `{"Result":{"Token":"rotated-old-account","RefreshToken":"rotated-refresh","TokenExpireAt":%d}}`, time.Now().Add(time.Hour).UnixMilli())
	}))
	defer srv.Close()
	b := NewBridge(Config{Token: "expired", RefreshToken: "old-refresh", ExpiresAt: time.Now().Add(-time.Hour).Format(time.RFC3339), AuthURL: srv.URL})
	b.SetCredentialSaver(func(c Config) error { saved <- c.Token; return nil })
	refreshDone := make(chan error, 1)
	go func() { _, err := b.ensureCredentials(context.Background()); refreshDone <- err }()
	<-started
	switchDone := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() {
		switchDone <- b.ChangeAccount(ctx, func(c Config) (Config, error) {
			if c.Token != "rotated-old-account" {
				t.Error("switch preceded rotation")
			}
			select {
			case token := <-saved:
				if token != "rotated-old-account" {
					t.Error("wrong profile session saved")
				}
			default:
				t.Error("old account session was not saved before switching")
			}
			c.Token = "backup-account"
			return c, nil
		})
	}()
	select {
	case <-switchDone:
		close(release)
		t.Fatal("account switch did not wait for the active refresh")
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	if err := <-refreshDone; err != nil {
		t.Fatal(err)
	}
	if err := <-switchDone; err != nil {
		t.Fatal(err)
	}
	if b.Config().Token != "backup-account" {
		t.Fatal("switch failed")
	}
}
