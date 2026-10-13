package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/nicokaka/mypokechamp_server/internal/crypto"
	"github.com/nicokaka/mypokechamp_server/internal/model"
	"github.com/nicokaka/mypokechamp_server/internal/store"
	"github.com/nicokaka/mypokechamp_server/internal/ws"
)

type Server struct {
	Store     *store.Store
	WSHandler *ws.Handler
	PublicURL string
	WSURL     string
}

func NewServer(s *store.Store, publicURL, wsURL string) *Server {
	return &Server{
		Store:     s,
		WSHandler: ws.NewHandler(s),
		PublicURL: publicURL,
		WSURL:     wsURL,
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/admin", s.handleAdminUI)
	mux.HandleFunc("/admin/api/state", s.handleAdminState)
	mux.HandleFunc("/admin/api/pokemon", s.handleAdminAddPokemon)
	mux.Handle("/ws", s.WSHandler)
	mux.Handle("/battle/ws", s.WSHandler)

	// Catch-all for /auth/* and /api/*
	mux.HandleFunc("/", s.handleGameRequest)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	st := s.Store.GetState()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "ok",
		"service":        "mypokechamp_server",
		"version":        "1.2.2-3859",
		"trainer_name":   st.Profile.Nam,
		"pokemon_in_box": len(st.Pokemons),
		"bots_loaded":    len(st.Bots),
		"time":           time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) decodeRequestPmp(r *http.Request) (model.ApiRequestEnvelope, map[string]interface{}) {
	var env model.ApiRequestEnvelope
	pmpMap := map[string]interface{}{}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil || len(bodyBytes) == 0 {
		return env, pmpMap
	}
	_ = json.Unmarshal(bodyBytes, &env)

	if len(env.Pmp) > 0 && string(env.Pmp) != "null" {
		_ = json.Unmarshal(env.Pmp, &pmpMap)
		return env, pmpMap
	}

	if env.Pmc != "" {
		dmy := r.Header.Get("X-PKB-DMY-VAL")
		st := s.Store.GetState()
		var dec string
		var err error
		if strings.HasPrefix(r.URL.Path, "/auth/") {
			dec, err = crypto.DecryptPmcAuth(env.Pmc, dmy)
		} else {
			sid := env.Sid
			if sid == "" {
				sid = st.SessionID
			}
			dec, err = crypto.DecryptPmcApi(env.Pmc, dmy, st.CsrfToken, sid, st.UdVer)
		}
		if err == nil && dec != "" {
			log.Printf("  └─ [CRYPTO] Decrypted pmc (%d chars): %s", len(dec), truncate(dec, 180))
			_ = json.Unmarshal([]byte(dec), &pmpMap)
		} else {
			log.Printf("  └─ [CRYPTO] pmc recebido (len=%d, dmy=%s) - usando modo híbrido", len(env.Pmc), dmy)
		}
	}
	return env, pmpMap
}

func (s *Server) writeGameResponse(w http.ResponseWriter, r *http.Request, pmp interface{}) {
	nowUnix := time.Now().Unix()
	dmy := r.Header.Get("X-PKB-DMY-VAL")
	if dmy == "" {
		dmy = fmt.Sprintf("%d", nowUnix)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-PKB-S-TIME", fmt.Sprintf("%d", nowUnix))
	w.Header().Set("X-PKB-DMY-VAL", dmy)

	tmv := uint32(1)
	ppv := uint32(1)
	wvv := uint32(1)
	sexp := 86400

	resp := model.ApiResponseEnvelope{
		Code:  0,
		TmV:   &tmv,
		PpV:   &ppv,
		WvV:   &wvv,
		Sexp:  &sexp,
		InH:   "inh_champ_2026",
		Asset: &model.ResponseAssetBundle{Ver: "1.2.2", Dom: s.PublicURL},
		Pmp:   pmp,
	}

	// Optional pmc encryption if client explicitly forces encrypted response
	if r.URL.Query().Get("encrypt") == "1" || r.Header.Get("X-PKB-FORCE-ENCRYPT") == "1" {
		if b, err := json.Marshal(pmp); err == nil {
			st := s.Store.GetState()
			var enc string
			if strings.HasPrefix(r.URL.Path, "/auth/") {
				enc, _ = crypto.EncryptPmpAuth(string(b), dmy, nowUnix)
			} else {
				enc, _ = crypto.EncryptPmpApi(string(b), dmy, st.CsrfToken, st.SessionID, st.UdVer)
			}
			if enc != "" {
				w.Header().Set("X-PKB-ENCRYPT", "1")
				resp.Pmc = enc
			}
		}
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleGameRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	path := r.URL.Path
	log.Printf("[API] %s %s from %s (DMY=%s, ENC=%s)",
		r.Method, path, r.RemoteAddr, r.Header.Get("X-PKB-DMY-VAL"), r.Header.Get("X-PKB-ENCRYPT"))

	env, pmpReq := s.decodeRequestPmp(r)
	st := s.Store.GetState()
	now := uint64(time.Now().Unix())

	switch path {
	case "/auth/get-token":
		uuid, _ := pmpReq["uuid"].(string)
		if uuid != "" {
			st = s.Store.UpdateAuth(uuid, "", st.CsrfToken, "")
		}
		s.writeGameResponse(w, r, model.AuthGetTokenPmp{
			Token: st.CsrfToken,
			Exp:   now + 86400,
			EnvLs: []model.AuthEnv{
				{Name: "Production", Url: s.PublicURL},
				{Name: "Stg", Url: s.PublicURL},
				{Name: "Req", Url: s.PublicURL},
			},
		})

	case "/auth/check-register-limit":
		s.writeGameResponse(w, r, map[string]interface{}{
			"isLimit": 0,
			"UD":      s.Store.BuildFullUD(),
		})

	case "/auth/register-user":
		uuid, _ := pmpReq["uuid"].(string)
		name, _ := pmpReq["name"].(string)
		st = s.Store.UpdateAuth(uuid, name, st.CsrfToken, st.SessionID)
		s.writeGameResponse(w, r, model.AuthRegisterUserPmp{
			Uuid:   st.UUID,
			Supid:  st.SupportID,
			Dvcode: st.DeviceCd,
			Lista:  []interface{}{},
			UD:     s.Store.BuildFullUD(),
		})

	case "/auth/login", "/auth/competition-mode-login":
		uuid, _ := pmpReq["uuid"].(string)
		st = s.Store.UpdateAuth(uuid, "", st.CsrfToken, st.SessionID)
		zero := 0
		s.writeGameResponse(w, r, model.AuthLoginPmp{
			Sid:      st.SessionID,
			Sexp:     86400,
			Refnotif: []interface{}{},
			Tmoff:    model.LoginTimeOffset{Offset: 0, Isfix: 0},
			Phtrbx:   &zero,
			Phtrul:   &zero,
			Btl:      0,
			Lby:      0,
			UD:       s.Store.BuildFullUD(),
		})

	case "/api/common/get-all-ud":
		s.writeGameResponse(w, r, map[string]interface{}{
			"UD": s.Store.BuildFullUD(),
		})

	case "/api/common/regular-check":
		s.writeGameResponse(w, r, model.CommonRegularCheckPmp{
			Exmsg:  0,
			Exmsgc: 0,
			Frireq: 0,
			Eveids: []string{"season_01"},
			Comids: []string{},
			Comntf: model.RegularCheckNotification{Wldq: 0, Eve: 0},
			Phtrbx: 0,
			Ban:    map[string]interface{}{},
			Revd:   0,
			UD:     s.Store.BuildFullUD(),
		})

	case "/api/training/update":
		uid, _ := pmpReq["uid"].(string)
		hp := byte(numOrZero(pmpReq["hp"]))
		atk := byte(numOrZero(pmpReq["atk"]))
		def := byte(numOrZero(pmpReq["def"]))
		agi := byte(numOrZero(pmpReq["agi"]))
		spatk := byte(numOrZero(pmpReq["spatk"]))
		spdef := byte(numOrZero(pmpReq["spdef"]))
		toku := uint16(numOrZero(pmpReq["toku"]))
		sei := byte(numOrZero(pmpReq["sei"]))
		var waza []uint16
		if wList, ok := pmpReq["waza"].([]interface{}); ok {
			for _, wItem := range wList {
				waza = append(waza, uint16(numOrZero(wItem)))
			}
		}
		ud := s.Store.ApplyTraining(uid, hp, atk, def, agi, spatk, spdef, toku, sei, waza)
		s.writeGameResponse(w, r, map[string]interface{}{
			"UD": ud,
		})

	case "/api/team/update":
		tid, _ := pmpReq["tid"].(string)
		nam, _ := pmpReq["nam"].(string)
		var mem []string
		if mList, ok := pmpReq["mem"].([]interface{}); ok {
			for _, m := range mList {
				if ms, ok := m.(string); ok {
					mem = append(mem, ms)
				}
			}
		}
		ud := s.Store.UpdateTeam(tid, model.ApiUserTeamParam{
			Tid:    tid,
			Nam:    nam,
			Lng:    2,
			Mem:    mem,
			Cuflg:  1,
			Dcuflg: 1,
		})
		s.writeGameResponse(w, r, map[string]interface{}{
			"UD": ud,
		})

	case "/api/battle/data/get-menu-data":
		s.writeGameResponse(w, r, map[string]interface{}{
			"seas": []interface{}{
				map[string]interface{}{
					"etyp": 1,
					"ssn":  1,
					"st":   now - 86400,
					"ed":   now + 2592000,
				},
			},
			"hist": []interface{}{},
			"UD":   s.Store.BuildFullUD(),
		})

	case "/api/battle/regulation/get":
		s.writeGameResponse(w, r, map[string]interface{}{
			"regid": "reg_standard_01",
			"reg": map[string]interface{}{
				"regid":  "reg_standard_01",
				"pnlist": []int{},
				"wzlist": []int{},
				"itlist": []int{},
			},
			"UD": s.Store.BuildFullUD(),
		})

	case "/api/battle/regulation/verify":
		s.writeGameResponse(w, r, map[string]interface{}{
			"errs":  []interface{}{},
			"derrs": map[string]interface{}{},
			"UD":    s.Store.BuildFullUD(),
		})

	case "/api/battle/start-matching":
		s.writeGameResponse(w, r, model.BattleStartMatchingPmp{
			Ws:   s.WSURL,
			Atk:  "atk_bot_match_001",
			Exp:  now + 3600,
			Ehsh: "ehsh_bot_match_001",
			UD:   s.Store.BuildFullUD(),
		})

	case "/api/battle/start-battle":
		s.writeGameResponse(w, r, model.BattleStartBattlePmp{
			Rmid:   "rm_bot_champion_001",
			Sprule: 0,
			UD:     s.Store.BuildFullUD(),
		})

	case "/api/battle/end-battle", "/api/battle/surrender-battle":
		s.writeGameResponse(w, r, model.BattleEndBattlePmp{
			Bca: 1,
			Rul: 1,
			Rw:  []interface{}{},
			UD:  s.Store.BuildFullUD(),
		})

	default:
		// Universal handler for all other /api/* and /auth/* routes
		_ = env
		s.writeGameResponse(w, r, map[string]interface{}{
			"UD": s.Store.BuildFullUD(),
		})
	}

	log.Printf("  └─ Respondido 200 OK em %v", time.Since(start))
}

func (s *Server) handleAdminState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(s.Store.GetState())
}

func (s *Server) handleAdminAddPokemon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		UID      string   `json:"uid"`
		MonsNo   uint16   `json:"mons_no"`
		Nickname string   `json:"nickname"`
		Ability  uint16   `json:"ability"`
		Item     uint16   `json:"item"`
		Nature   byte     `json:"nature"`
		Shiny    byte     `json:"shiny"`
		TeraType int      `json:"tera_type"`
		Moves    []uint16 `json:"moves"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Moves) == 0 {
		req.Moves = []uint16{85, 89, 53, 182}
	}
	p := model.ApiUserPokemonParam{
		Ver: 1,
		B1:  req.MonsNo,
		B4:  125000,
		B5:  req.Ability,
		B6:  req.Moves,
		B7:  req.Item,
		B8:  4,
		B9:  4,
		Ba:  252,
		Bc:  252,
		Bj:  req.Nature,
		Bk:  req.Nature,
		Bl:  req.Shiny,
		Bm:  req.TeraType,
		Bn:  2,
		Bo:  req.Nickname,
	}
	s.Store.AddOrUpdatePokemon(req.UID, p)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (s *Server) handleAdminUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminHTML))
}

func numOrZero(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	default:
		return 0
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

const adminHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
<meta charset="UTF-8">
<title>MyPokeChamp Private Server - Painel Admin</title>
<style>
  body { background: #0f172a; color: #f8fafc; font-family: system-ui, sans-serif; margin: 0; padding: 24px; }
  .container { max-width: 1000px; margin: 0 auto; }
  h1 { color: #38bdf8; margin-bottom: 4px; }
  .card { background: #1e293b; border: 1px solid #334155; border-radius: 12px; padding: 20px; margin-top: 20px; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 12px; margin-top: 12px; }
  .poke { background: #0f172a; border: 1px solid #475569; border-radius: 8px; padding: 12px; }
  .shiny { border-color: #facc15; }
  input, button { padding: 8px 12px; border-radius: 6px; border: 1px solid #475569; background: #0f172a; color: #fff; margin: 4px; }
  button { background: #0284c7; border: none; cursor: pointer; font-weight: bold; }
  button:hover { background: #0369a1; }
</style>
</head>
<body>
<div class="container">
  <h1>🏆 MyPokeChamp Private Server (v1.2.2-3859)</h1>
  <p>Gerenciador de Box Pokémon, Times Competitivos e Bots</p>
  <div class="card">
    <h3>➕ Injetar Pokémon Competitivo na Box</h3>
    <input id="mons_no" type="number" placeholder="Dex ID (ex: 6 = Charizard)" value="150">
    <input id="nickname" type="text" placeholder="Nome / Nickname" value="Mewtwo">
    <input id="ability" type="number" placeholder="Ability ID (ex: 46)" value="46">
    <input id="item" type="number" placeholder="Item ID (ex: 270)" value="270">
    <input id="shiny" type="number" placeholder="Shiny (1=Sim, 0=Não)" value="1">
    <button onclick="addPoke()">Injetar na Box</button>
  </div>
  <div class="card">
    <h3>📦 Pokémon na Box (<span id="cnt">0</span>)</h3>
    <div id="box" class="grid"></div>
  </div>
</div>
<script>
async function load() {
  const res = await fetch('/admin/api/state');
  const st = await res.json();
  const pokes = Object.values(st.pokemons || {});
  document.getElementById('cnt').innerText = pokes.length;
  document.getElementById('box').innerHTML = pokes.map(p =>
    '<div class="poke ' + (p.bl ? 'shiny' : '') + '">' +
    '<b>#' + p.b1 + ' ' + p.bo + (p.bl ? ' ✨' : '') + '</b><br>' +
    '<small>UID: ' + p.a1 + ' | Item: ' + p.b7 + '<br>Moves: ' + (p.b6 || []).join(', ') + '</small>' +
    '</div>'
  ).join('');
}
async function addPoke() {
  await fetch('/admin/api/pokemon', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
      mons_no: parseInt(document.getElementById('mons_no').value),
      nickname: document.getElementById('nickname').value,
      ability: parseInt(document.getElementById('ability').value),
      item: parseInt(document.getElementById('item').value),
      shiny: parseInt(document.getElementById('shiny').value),
      nature: 15,
      tera_type: 1,
      moves: [94, 58, 85, 182]
    })
  });
  load();
}
load();
</script>
</body>
</html>`
