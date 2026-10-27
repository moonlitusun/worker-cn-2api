package workbuddy

import (
	"strings"
	"testing"
)

func TestDeriveDeviceID(t *testing.T) {
	id1 := DeriveDeviceID("user-123456")
	id2 := DeriveDeviceID("user-123456")
	if id1 != id2 {
		t.Errorf("DeriveDeviceID must be deterministic, got %s vs %s", id1, id2)
	}

	if !strings.HasPrefix(id1, "wb-") {
		t.Errorf("expected derive_id to start with wb-, got %s", id1)
	}

	idOther := DeriveDeviceID("user-999999")
	if id1 == idOther {
		t.Errorf("different users must have distinct derive_ids")
	}
}

func TestBuildUpstreamRequest(t *testing.T) {
	reqBody := map[string]interface{}{
		"model": "deepseek-v3",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "hello"},
		},
		"stream":      true,
		"temperature": 0.5,
	}

	req, err := BuildUpstreamRequest(reqBody, "deepseek-v3", "wb-device-123")
	if err != nil {
		t.Fatalf("BuildUpstreamRequest failed: %v", err)
	}

	if req.DeriveID != "wb-device-123" {
		t.Errorf("derive_id mismatch: %s", req.DeriveID)
	}
	if len(req.Messages) != 1 {
		t.Errorf("messages count mismatch: %d", len(req.Messages))
	}
	if req.Options["enable_reasoning"] != true {
		t.Errorf("expected enable_reasoning to be true")
	}
}

func TestBuildUpstreamRequestHonorsDisabledThinking(t *testing.T) {
	req, err := BuildUpstreamRequest(map[string]interface{}{
		"thinking":    map[string]interface{}{"type": "disabled"},
		"temperature": 0.2, "max_tokens": float64(256),
	}, "deepseek-r1", "test-device")
	if err != nil {
		t.Fatal(err)
	}
	if req.Options["enable_reasoning"] != false {
		t.Fatalf("reasoning was forced on: %v", req.Options)
	}
	if req.Options["temperature"] != 0.2 || req.Options["max_tokens"] != float64(256) {
		t.Fatal("existing controls lost")
	}
}
