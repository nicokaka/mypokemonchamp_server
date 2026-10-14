package api_test

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
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
	srv := api.NewServer(st, "http://164.152.40.114:8080", "ws://164.152.40.114:8080/ws")
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
	// Verify Asset is nil so client AssetBundleDomain is never overwritten
	if envResp.Asset != nil {
		t.Fatalf("expected Asset to be nil in response envelope, got %+v", envResp.Asset)
	}

	// 2. Test /auth/register-user and verify lista is a JSON object {} and Pokemon bf is a JSON array [31,31,31,31,31,31]
	reqReg := httptest.NewRequest(http.MethodPost, "/auth/register-user", bytes.NewReader([]byte(`{"clV":"1.2.2","pmp":{"uuid":"test-uuid-777","name":"Nicolas"}}`)))
	recReg := httptest.NewRecorder()
	mux.ServeHTTP(recReg, reqReg)
	if recReg.Code != http.StatusOK {
		t.Fatalf("/auth/register-user status = %d", recReg.Code)
	}
	rawJSON := recReg.Body.String()
	if !strings.Contains(rawJSON, `"bf":[31,31,31,31,31,31]`) {
		t.Fatalf("expected Pokemon IVs bf to serialize as integer array [31,31,31,31,31,31], got: %s", rawJSON)
	}
	if !strings.Contains(rawJSON, `"lista":{}`) {
		t.Fatalf("expected register-user lista to serialize as JSON object {}, got: %s", rawJSON)
	}
	if !strings.Contains(rawJSON, `"trbad":["1"]`) {
		t.Fatalf("expected profile trbad to serialize as string array, got: %s", rawJSON)
	}

	// 3. Test hyphenated battle routes
	for _, route := range []string{
		"/api/common/get-all-ud",
		"/api/common/regular-check",
		"/api/battle-data/get-menu-data",
		"/api/battle-regulation/get",
		"/api/battle-regulation/verify",
		"/api/battle-regulation/private-lobby-list",
		"/api/battle/start-matching",
		"/api/battle/start-battle",
		"/api/battle/end-battle",
		"/api/team/update",
		"/api/training/update",
	} {
		r := httptest.NewRequest(http.MethodPost, route, bytes.NewReader([]byte(`{"clV":"1.2.2"}`)))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s status = %d", route, w.Code)
		}
	}
}

func TestWebSocket2BattleFlow(t *testing.T) {
	st, err := store.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	srv := api.NewServer(st, "http://127.0.0.1:8080", "ws://127.0.0.1:8080/ws")
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/ws", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	conn, err := net.Dial("tcp", strings.TrimPrefix(ts.URL, "http://"))
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer conn.Close()

	if err := req.Write(conn); err != nil {
		t.Fatalf("write handshake: %v", err)
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, req)
	if err != nil || resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("unexpected handshake response: %v, err=%v", resp, err)
	}

	writeWS := func(v interface{}) {
		b, _ := json.Marshal(v)
		hdr := []byte{0x81, byte(len(b))}
		_, _ = conn.Write(hdr)
		_, _ = conn.Write(b)
	}
	readWS := func() map[string]interface{} {
		hdr := make([]byte, 2)
		if _, err := io.ReadFull(br, hdr); err != nil {
			t.Fatalf("readWS hdr: %v", err)
		}
		length := uint64(hdr[1] & 0x7f)
		if length == 126 {
			ext := make([]byte, 2)
			_, _ = io.ReadFull(br, ext)
			length = uint64(binary.BigEndian.Uint16(ext))
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(br, payload); err != nil {
			t.Fatalf("readWS payload: %v", err)
		}
		var m map[string]interface{}
		if err := json.Unmarshal(payload, &m); err != nil {
			t.Fatalf("unmarshal ws json: %v", err)
		}
		return m
	}

	// 1. Send send_start_matching -> expect res_start_matching, send_match_ready2, send_match_success
	writeWS(map[string]interface{}{"action": "send_start_matching", "cnt": "1"})
	m1 := readWS()
	m2 := readWS()
	m3 := readWS()
	if m1["action"] != "res_start_matching" || m2["action"] != "send_match_ready2" || m3["action"] != "send_match_success" {
		t.Fatalf("unexpected matching sequence: %v, %v, %v", m1["action"], m2["action"], m3["action"])
	}
	if _, ok := m3["cnt"].(string); !ok {
		t.Fatalf("expected cnt to be string, got %T", m3["cnt"])
	}

	// 2. Send command (Turn 1 Entry) -> expect res_command with Sequences containing Cmd=2 (CtrlEntry)
	writeWS(map[string]interface{}{
		"action": "command",
		"cnt":    "2",
		"dp":     map[string]interface{}{"bl": `{"CommandTypeNum":1,"UserIndex":0,"PokeIndex":0}`},
	})
	cmd1 := readWS()
	if cmd1["action"] != "res_command" {
		t.Fatalf("expected res_command, got %v", cmd1["action"])
	}
	dp1 := cmd1["dp"].(map[string]interface{})
	if !strings.Contains(dp1["bl"].(string), `"Sequences"`) {
		t.Fatalf("expected Sequences in dp.bl, got %s", dp1["bl"])
	}

	// 3. Send command (Turn 2 Move) -> expect res_command with et=1 and CtrlBattleEnd (Cmd=3)
	writeWS(map[string]interface{}{
		"action": "command",
		"cnt":    "3",
		"dp":     map[string]interface{}{"bl": `{"CommandTypeNum":8,"Move":89}`},
	})
	cmd2 := readWS()
	dp2 := cmd2["dp"].(map[string]interface{})
	if int(dp2["et"].(float64)) != 1 {
		t.Fatalf("expected et=1 on winning turn, got %v", dp2["et"])
	}
}

func TestAdminDashboardAndMutations(t *testing.T) {
	st, err := store.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	srv := api.NewServer(st, "https://127.0.0.1:9443", "wss://127.0.0.1:9443/ws")
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// 1. GET /admin & GET /accounts & GET / should return HTML dashboard
	for _, p := range []string{"/", "/admin", "/accounts"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "MyPokeChamp Studio") {
			t.Fatalf("GET %s failed: code=%d", p, rec.Code)
		}
	}

	// 2. POST /admin/api/profile
	prfBody := `{"nam":"AshKetchum","vp":777777,"rank_single":10,"rating_single":1999}`
	reqPrf := httptest.NewRequest(http.MethodPost, "/admin/api/profile", strings.NewReader(prfBody))
	recPrf := httptest.NewRecorder()
	mux.ServeHTTP(recPrf, reqPrf)
	if recPrf.Code != http.StatusOK || st.GetState().Profile.Nam != "AshKetchum" || st.GetState().Main.Vp != 777777 {
		t.Fatalf("POST /admin/api/profile failed: code=%d", recPrf.Code)
	}

	// 3. POST /admin/api/unlock-meta
	reqMeta := httptest.NewRequest(http.MethodPost, "/admin/api/unlock-meta", nil)
	recMeta := httptest.NewRecorder()
	mux.ServeHTTP(recMeta, reqMeta)
	if recMeta.Code != http.StatusOK || len(st.GetState().Pokemons) < 20 {
		t.Fatalf("POST /admin/api/unlock-meta failed: code=%d, count=%d", recMeta.Code, len(st.GetState().Pokemons))
	}

	// 4. POST /admin/api/maxout
	reqMax := httptest.NewRequest(http.MethodPost, "/admin/api/maxout", nil)
	recMax := httptest.NewRecorder()
	mux.ServeHTTP(recMax, reqMax)
	if recMax.Code != http.StatusOK || st.GetState().Main.Vp != 9999999 {
		t.Fatalf("POST /admin/api/maxout failed: code=%d", recMax.Code)
	}
}

