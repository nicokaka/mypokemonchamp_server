package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicokaka/mypokechamp_server/internal/crypto"
	"github.com/nicokaka/mypokechamp_server/internal/model"
	"github.com/nicokaka/mypokechamp_server/internal/store"
)

func TestMasterDataEndToEnd(t *testing.T) {
	st, err := store.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("store.NewStore failed: %v", err)
	}
	srv := NewServer(st, "https://164.152.40.114:443", "wss://164.152.40.114:443/ws")
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// 1. Call POST /auth/get-token and inspect response.Md
	req := httptest.NewRequest(http.MethodPost, "https://164.152.40.114:443/auth/get-token", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /auth/get-token, got %d", rec.Code)
	}

	var env model.ApiResponseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if env.Md == nil {
		t.Fatalf("expected non-nil env.Md")
	}

	mdListMeta := env.Md
	// Simulate Android client (platformCode = 3)
	platformCode := 3
	listFn, listKey, listIV := deriveMdMetaListInfo(platformCode, mdListMeta.Ver, mdListMeta.Tim, mdListMeta.Ra)
	if listFn != "eec18ea04378b5c189d3c9050aa8a01e08cb60d3" {
		t.Fatalf("expected Android listFn eec18ea04378b5c189d3c9050aa8a01e08cb60d3, got %s", listFn)
	}

	// 2. Fetch /md/c/{listFn}
	mdReq := httptest.NewRequest(http.MethodGet, "/md/c/"+listFn, nil)
	mdRec := httptest.NewRecorder()
	mux.ServeHTTP(mdRec, mdReq)
	if mdRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /md/c/%s, got %d", listFn, mdRec.Code)
	}

	rawListBytes := mdRec.Body.Bytes()
	if md5HexBytes(rawListBytes) != mdListMeta.Lchs {
		t.Fatalf("Lchs mismatch: got %s, want %s", md5HexBytes(rawListBytes), mdListMeta.Lchs)
	}
	if len(rawListBytes) != mdListMeta.Lsiz {
		t.Fatalf("Lsiz mismatch: got %d, want %d", len(rawListBytes), mdListMeta.Lsiz)
	}

	// Decrypt MdMetaList exactly as client's DecryptAndDeserialize<MdMetaList> does
	cipherBytes, err := base64.StdEncoding.DecodeString(string(rawListBytes))
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}
	plainBytes, err := crypto.AesDecrypt(cipherBytes, []byte(listKey), []byte(listIV))
	if err != nil {
		t.Fatalf("AesDecrypt MdMetaList failed: %v", err)
	}
	if crypto.MD5Hex(string(plainBytes)) != mdListMeta.Ljhs {
		t.Fatalf("Ljhs mismatch: got %s, want %s", crypto.MD5Hex(string(plainBytes)), mdListMeta.Ljhs)
	}

	var metaList model.MdMetaList
	if err := json.Unmarshal(plainBytes, &metaList); err != nil {
		t.Fatalf("unmarshal MdMetaList failed: %v", err)
	}
	if len(metaList.List) != len(mdAllTableNames) {
		t.Fatalf("expected %d tables in MdMetaList, got %d", len(mdAllTableNames), len(metaList.List))
	}

	// 3. Verify name decryption and table download for every table
	for i := range metaList.List {
		m := &metaList.List[i]
		tim := m.Tim
		if tim <= 0 {
			tim = mdListMeta.Tim
		}
		m.Name = decryptMdMetaListName(m.Nse, tim)
		if m.Name != mdAllTableNames[i] {
			t.Fatalf("table[%d] decrypted name = %q, want %q", i, m.Name, mdAllTableNames[i])
		}

		fn, key, iv := deriveMdMetaInfo(platformCode, m.Name, m.Ver, tim, m.Ra)
		tReq := httptest.NewRequest(http.MethodGet, "/md/c/"+fn, nil)
		tRec := httptest.NewRecorder()
		mux.ServeHTTP(tRec, tReq)
		if tRec.Code != http.StatusOK {
			t.Fatalf("table %s GET /md/c/%s returned %d", m.Name, fn, tRec.Code)
		}
		tBytes := tRec.Body.Bytes()
		if md5HexBytes(tBytes) != m.Chs {
			t.Fatalf("table %s chs mismatch", m.Name)
		}
		tCipher, err := base64.StdEncoding.DecodeString(string(tBytes))
		if err != nil {
			t.Fatalf("table %s base64 decode failed: %v", m.Name, err)
		}
		tPlain, err := crypto.AesDecrypt(tCipher, []byte(key), []byte(iv))
		if err != nil {
			t.Fatalf("table %s AesDecrypt failed: %v", m.Name, err)
		}
		if crypto.MD5Hex(string(tPlain)) != m.Jhs {
			t.Fatalf("table %s jhs mismatch", m.Name)
		}
	}
}
