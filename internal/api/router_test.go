package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nicokaka/mypokechamp_server/internal/api"
	"github.com/nicokaka/mypokechamp_server/internal/crypto"
	"github.com/nicokaka/mypokechamp_server/internal/model"
	"github.com/nicokaka/mypokechamp_server/internal/store"
)

func TestCryptoRoundTrip(t *testing.T) {
	now := time.Now().Unix()
	dmy := "1234567890123"
	plainAuth := `{"uuid":"champ-uuid-001","hash":"abc","skipHashValidation":1}`

	pmcAuth, err := crypto.EncryptPmpAuth(plainAuth, dmy, now)
	if err != nil {
		t.Fatalf("EncryptPmpAuth failed: %v", err)
	}
	decAuth, err := crypto.DecryptPmcAuth(pmcAuth, dmy)
	if err != nil {
		t.Fatalf("DecryptPmcAuth failed: %v", err)
	}
	if decAuth != plainAuth {
		t.Fatalf("Auth roundtrip mismatch: got %q, want %q", decAuth, plainAuth)
	}

	plainApi := `{"tgt":[1,2,3],"udVer":1}`
	pmcApi, err := crypto.EncryptPmpApi(plainApi, dmy, "tok_01", "sid_01", 1)
	if err != nil {
		t.Fatalf("EncryptPmpApi failed: %v", err)
	}
	decApi, err := crypto.DecryptPmcApi(pmcApi, dmy, "tok_01", "sid_01", 1)
	if err != nil {
		t.Fatalf("DecryptPmcApi failed: %v", err)
	}
	if decApi != plainApi {
		t.Fatalf("API roundtrip mismatch: got %q, want %q", decApi, plainApi)
	}
}

func TestFullBootAndBattleFlow(t *testing.T) {
	st, err := store.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	srv := api.NewServer(st, "https://164.152.40.114:9443", "wss://164.152.40.114:9443/ws")
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// 1. Test /auth/get-token with encrypted pmc
	dmy := "9876543210"
	pmc, _ := crypto.EncryptPmpAuth(`{"uuid":"test-uuid-777","skipHashValidation":1}`, dmy, time.Now().Unix())
	reqBody, _ := json.Marshal(model.ApiRequestEnvelope{
		ClV: "1.2.2",
		Pmc: pmc,
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/get-token", bytes.NewReader(reqBody))
	req.Header.Set("X-PKB-DMY-VAL", dmy)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("/auth/get-token status = %d", rec.Code)
	}
	var envResp model.ApiResponseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envResp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if envResp.Code != 0 {
		t.Fatalf("expected code 0, got %d", envResp.Code)
	}

	// 2. Test /api/common/get-all-ud
	req2 := httptest.NewRequest(http.MethodPost, "/api/common/get-all-ud", bytes.NewReader([]byte(`{"clV":"1.2.2"}`)))
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("/api/common/get-all-ud status = %d", rec2.Code)
	}

	// 3. Test /api/battle/start-matching
	req3 := httptest.NewRequest(http.MethodPost, "/api/battle/start-matching", bytes.NewReader([]byte(`{"clV":"1.2.2"}`)))
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("/api/battle/start-matching status = %d", rec3.Code)
	}
}
