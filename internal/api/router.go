package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nicokaka/mypokechamp_server/internal/crypto"
	"github.com/nicokaka/mypokechamp_server/internal/model"
	"github.com/nicokaka/mypokechamp_server/internal/store"
	"github.com/nicokaka/mypokechamp_server/internal/ws"
)

type Server struct {
	Store      *store.Store
	WSHandler  *ws.Handler
	PublicURL  string
	WSURL      string
	trafficMu  sync.Mutex
	trafficSeq int64
	trafficLog []TrafficEntry
}

func NewServer(s *store.Store, publicURL, wsURL string) *Server {
	return &Server{
		Store:     s,
		WSHandler: ws.NewHandler(s),
		PublicURL: publicURL,
		WSURL:     wsURL,
	}
}

func (s *Server) resolvePublicURL(r *http.Request) string {
	if r != nil && r.Host != "" {
		scheme := "http"
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			scheme = "https"
		}
		return scheme + "://" + r.Host
	}
	return s.PublicURL
}

func (s *Server) resolveWSURL(r *http.Request) string {
	if r != nil && r.Host != "" {
		scheme := "ws"
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			scheme = "wss"
		}
		return scheme + "://" + r.Host + "/ws"
	}
	return s.WSURL
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/admin", s.handleAdminUI)
	mux.HandleFunc("/accounts", s.handleAdminUI)
	mux.HandleFunc("/admin/api/state", s.handleAdminState)
	mux.HandleFunc("/admin/api/traffic", s.handleAdminTraffic)
	mux.HandleFunc("/admin/api/profile", s.handleAdminUpdateProfile)
	mux.HandleFunc("/admin/api/avatar", s.handleAdminUpdateAvatar)
	mux.HandleFunc("/admin/api/maxout", s.handleAdminMaxOut)
	mux.HandleFunc("/admin/api/unlock-meta", s.handleAdminUnlockMeta)
	mux.HandleFunc("/admin/api/pokemon", s.handleAdminAddPokemon)
	mux.HandleFunc("/admin/api/pokemon/delete", s.handleAdminDeletePokemon)
	mux.HandleFunc("/admin/api/team", s.handleAdminSaveTeam)
	mux.HandleFunc("/admin/api/team/delete", s.handleAdminDeleteTeam)
	mux.HandleFunc("/admin/api/bot", s.handleAdminSaveBot)
	mux.HandleFunc("/admin/api/bot/delete", s.handleAdminDeleteBot)
	mux.HandleFunc("/admin/api/reset", s.handleAdminReset)
	mux.Handle("/ws", s.WSHandler)
	mux.Handle("/battle/ws", s.WSHandler)
	mux.HandleFunc("/md/", s.handleMasterData)

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
			dec, err = crypto.DecryptPmcAuth(env.Pmc, dmy, env.Dmy)
		} else {
			sid := env.Sid
			if sid == "" {
				sid = st.SessionID
			}
			dec, err = crypto.DecryptPmcApi(env.Pmc, dmy, st.CsrfToken, sid, 1, int(st.UdVer))
			if err != nil && env.Dmy != "" {
				dec, err = crypto.DecryptPmcApi(env.Pmc, env.Dmy, st.CsrfToken, sid, 1, int(st.UdVer))
			}
		}
		if err == nil && dec != "" {
			log.Printf("  └─ [CRYPTO] Decrypted pmc (%d chars): %s", len(dec), truncate(dec, 180))
			_ = json.Unmarshal([]byte(dec), &pmpMap)
		} else {
			log.Printf("  └─ [CRYPTO] pmc recebido (len=%d, hdr_dmy=%s, env_dmy=%s) - usando modo híbrido", len(env.Pmc), dmy, env.Dmy)
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

	sTimeStr := fmt.Sprintf("%d", nowUnix)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-PKB-S-TIME", sTimeStr)
	w.Header().Set("X-PKB-DMY-VAL", dmy)

	tmv := uint32(1)
	ppv := uint32(1)
	wvv := uint32(1)
	sexp := 86400

	resp := model.ApiResponseEnvelope{
		Code: 0,
		Md:   s.buildMdListMeta(r),
		TmV:  &tmv,
		PpV:  &ppv,
		WvV:  &wvv,
		Sexp: &sexp,
		InH:  "inh_champ_2026",
		Pmp:  pmp,
	}

	// Always populate resp.Pmc unless explicitly disabled (?encrypt=0).
	if r.URL.Query().Get("encrypt") != "0" {
		if b, err := json.Marshal(pmp); err == nil {
			st := s.Store.GetState()
			var enc string
			if strings.HasPrefix(r.URL.Path, "/auth/") {
				enc, _ = crypto.EncryptPmpAuth(string(b), dmy, nowUnix)
			} else {
				enc, _ = crypto.EncryptPmpApi(string(b), dmy, st.CsrfToken, st.SessionID, 1)
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
	if r.Method == http.MethodGet && r.URL.Path == "/" {
		s.handleAdminUI(w, r)
		return
	}
	if r.Method != http.MethodPost && !strings.HasPrefix(r.URL.Path, "/auth/") && !strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
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
		if uuid == "" && len(env.Dmy) == 42 {
			uuid = env.Dmy
		}
		if uuid != "" {
			st = s.Store.UpdateAuth(uuid, "", st.CsrfToken, "")
		}
		pubURL := s.resolvePublicURL(r)
		s.writeGameResponse(w, r, model.AuthGetTokenPmp{
			Token: st.CsrfToken,
			Exp:   now + 86400,
			EnvLs: []model.AuthEnv{
				{Name: "Production", Url: pubURL},
				{Name: "Stg", Url: pubURL},
				{Name: "Req", Url: pubURL},
			},
		})

	case "/auth/check-register-limit":
		s.writeGameResponse(w, r, map[string]interface{}{
			"isLimit": 0,
			"UD":      s.Store.BuildFullUD(),
		})

	case "/auth/register-user":
		uuid, _ := pmpReq["uuid"].(string)
		if uuid == "" && len(env.Dmy) == 42 {
			uuid = env.Dmy
		}
		name, _ := pmpReq["name"].(string)
		st = s.Store.UpdateAuth(uuid, name, st.CsrfToken, st.SessionID)
		s.writeGameResponse(w, r, model.AuthRegisterUserPmp{
			Uuid:   st.UUID,
			Supid:  st.SupportID,
			Dvcode: st.DeviceCd,
			Lista:  map[string]byte{},
			UD:     s.Store.BuildFullUD(),
		})

	case "/auth/login":
		uuid, _ := pmpReq["uuid"].(string)
		if uuid == "" && len(env.Dmy) == 42 {
			uuid = env.Dmy
		}
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

	case "/auth/competition-mode-login":
		s.writeGameResponse(w, r, map[string]interface{}{
			"comid": "com_001",
			"com":   map[string]interface{}{},
			"ispar": 1,
			"entdt": map[string]interface{}{},
			"UD":    s.Store.BuildFullUD(),
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

	case "/api/user/update":
		var nam, trtit, trico string
		var trbad []string
		if prfMap, ok := pmpReq["prf"].(map[string]interface{}); ok {
			nam, _ = prfMap["nam"].(string)
			trtit, _ = prfMap["trtit"].(string)
			trico, _ = prfMap["trico"].(string)
			if tbList, ok := prfMap["trbad"].([]interface{}); ok {
				for _, item := range tbList {
					if sVal, ok := item.(string); ok {
						trbad = append(trbad, sVal)
					}
				}
			}
		} else {
			nam, _ = pmpReq["nam"].(string)
		}
		ud := s.Store.UpdateProfile(nam, trtit, trico, trbad)
		stNow := s.Store.GetState()
		s.writeGameResponse(w, r, map[string]interface{}{
			"prf": stNow.Profile,
			"UD":  ud,
		})

	case "/api/avatar/update", "/api/avatar/preset/update", "/api/avatar/preset/delete":
		var avat model.ApiUserAvatarParam
		if b, err := json.Marshal(pmpReq); err == nil {
			_ = json.Unmarshal(b, &avat)
		}
		ud := s.Store.UpdateAvatar(avat)
		s.writeGameResponse(w, r, map[string]interface{}{
			"UD": ud,
		})

	case "/api/training/update":
		// Supports both list: Dictionary<string, TrainingRequestUniqueParam> (Type 14001) and direct uid fields
		applyOne := func(m map[string]interface{}) {
			uid, _ := m["uid"].(string)
			if uid == "" {
				return
			}
			hp := byte(numOrZero(m["hp"]))
			atk := byte(numOrZero(m["atk"]))
			def := byte(numOrZero(m["def"]))
			agi := byte(numOrZero(m["agi"]))
			spatk := byte(numOrZero(m["spatk"]))
			spdef := byte(numOrZero(m["spdef"]))
			toku := uint16(numOrZero(m["toku"]))
			sei := byte(numOrZero(m["sei"]))
			var waza []int16
			if wList, ok := m["waza"].([]interface{}); ok {
				for _, wItem := range wList {
					waza = append(waza, int16(numOrZero(wItem)))
				}
			}
			_ = s.Store.ApplyTraining(uid, hp, atk, def, agi, spatk, spdef, toku, sei, waza)
		}
		if listMap, ok := pmpReq["list"].(map[string]interface{}); ok && len(listMap) > 0 {
			for k, v := range listMap {
				if itemMap, ok := v.(map[string]interface{}); ok {
					if _, hasUID := itemMap["uid"]; !hasUID {
						itemMap["uid"] = k
					}
					applyOne(itemMap)
				}
			}
		} else {
			applyOne(pmpReq)
		}
		s.writeGameResponse(w, r, map[string]interface{}{
			"UD": s.Store.BuildFullUD(),
		})

	case "/api/team/update":
		// Supports both ApiTeamUpdateRequestParam.UniqueParam.team (List<ApiUserTeamParam>, Type 13980)
		// and direct single-team fields (tid, nam, mem).
		if rawTeams, ok := pmpReq["team"]; ok {
			var teams []model.ApiUserTeamParam
			if b, err := json.Marshal(rawTeams); err == nil {
				_ = json.Unmarshal(b, &teams)
			}
			if len(teams) > 0 {
				ud := s.Store.UpdateTeams(teams)
				s.writeGameResponse(w, r, map[string]interface{}{
					"UD": ud,
				})
				break
			}
		}
		tid := numOrZero(pmpReq["tid"])
		if tid == 0 {
			if tidStr, ok := pmpReq["tid"].(string); ok {
				if parsed, err := strconv.Atoi(strings.TrimPrefix(tidStr, "team_0")); err == nil && parsed > 0 {
					tid = parsed
				} else {
					tid = 1
				}
			} else {
				tid = 1
			}
		}
		nam, _ := pmpReq["nam"].(string)
		if nam == "" {
			nam = fmt.Sprintf("Team %d", tid)
		}
		var mem []string
		if mList, ok := pmpReq["mem"].([]interface{}); ok {
			for _, m := range mList {
				if ms, ok := m.(string); ok {
					mem = append(mem, ms)
				}
			}
		}
		var itms []model.ApiUserTeamItemParam
		if iList, ok := pmpReq["itms"].([]interface{}); ok {
			if b, err := json.Marshal(iList); err == nil {
				_ = json.Unmarshal(b, &itms)
			}
		}
		ud := s.Store.UpdateTeam(tid, model.ApiUserTeamParam{
			Tid:    tid,
			Nam:    nam,
			Lng:    2,
			Mem:    mem,
			Cuflg:  1,
			Dcuflg: 1,
			Itms:   itms,
		})
		s.writeGameResponse(w, r, map[string]interface{}{
			"UD": ud,
		})

	case "/api/team/delete":
		tid := numOrZero(pmpReq["tid"])
		ud := s.Store.DeleteTeam(tid)
		s.writeGameResponse(w, r, map[string]interface{}{
			"UD": ud,
		})

	case "/api/box/nickname":
		uid, _ := pmpReq["uid"].(string)
		nickname, _ := pmpReq["bo"].(string)
		if nickname == "" {
			nickname, _ = pmpReq["nam"].(string)
		}
		ud := s.Store.UpdateNickname(uid, nickname)
		s.writeGameResponse(w, r, map[string]interface{}{
			"UD": ud,
		})

	case "/api/box/ribbon":
		uid, _ := pmpReq["uid"].(string)
		bw := byte(numOrZero(pmpReq["bw"]))
		ud := s.Store.UpdateRibbon(uid, bw)
		s.writeGameResponse(w, r, map[string]interface{}{
			"UD": ud,
		})

	case "/api/box/goodbye-check":
		s.writeGameResponse(w, r, map[string]interface{}{
			"ent": []string{},
			"UD":  s.Store.BuildFullUD(),
		})

	case "/api/box/goodbye":
		var uids []string
		if uList, ok := pmpReq["uids"].([]interface{}); ok {
			for _, u := range uList {
				if us, ok := u.(string); ok {
					uids = append(uids, us)
				}
			}
		}
		ud := s.Store.DeletePokemons(uids)
		s.writeGameResponse(w, r, map[string]interface{}{
			"UD": ud,
		})

	case "/api/battle-data/get-menu-data", "/api/battle/data/get-menu-data":
		// Net.ApiBattleDataGetMenuDataResponseParam.UniqueParam (Type 13577):
		// seas is Dictionary<int, Dictionary<byte, ApiBattleDataGetMenuDataSeasonsData>>
		s.writeGameResponse(w, r, map[string]interface{}{
			"seas": map[string]interface{}{
				"1": map[string]interface{}{
					"1": map[string]interface{}{
						"etyp":   byte(1),
						"eveid":  "season_01",
						"esta":   now - 86400,
						"eend":   now + 2592000,
						"serc":   "reg_standard_01",
						"serssn": 1,
						"agst":   1,
					},
				},
			},
			"hist": []interface{}{},
			"UD":   s.Store.BuildFullUD(),
		})

	case "/api/battle-regulation/get", "/api/battle/regulation/get":
		// Net.ApiBattleRegulationGetResponseParam.UniqueParam (Type 13589, 13591)
		s.writeGameResponse(w, r, map[string]interface{}{
			"reg": map[string]interface{}{
				"regid":  "reg_standard_01",
				"rtit":   "Standard Regulation A",
				"opsta":  now - 86400,
				"opend":  now + 2592000,
				"brule":  byte(1),
				"bamin":  900,
				"bamax":  1200,
				"limitp": 6,
				"avapl":  map[string]interface{}{"pnlist": [][]int{}},
				"ngwaza": map[string]interface{}{"wzlist": []int{}},
				"ngitem": map[string]interface{}{"itlist": []int{}},
				"btltyp": 1,
				"tobtim": 900,
				"platim": 60,
				"comtim": 45,
				"pkview": 1,
				"pkvtim": 60,
				"sprule": []map[string]interface{}{{"typ": byte(1), "use": byte(1)}},
				"pnick":  1,
				"titlev": 50,
			},
			"UD": s.Store.BuildFullUD(),
		})

	case "/api/battle-regulation/private-lobby-list", "/api/battle/regulation/private-lobby-list":
		// Net.ApiBattleRegulationPrivateLobbyListResponseParam.UniqueParam (Type 13598)
		s.writeGameResponse(w, r, map[string]interface{}{
			"regs": []interface{}{},
			"UD":   s.Store.BuildFullUD(),
		})

	case "/api/battle-regulation/verify", "/api/battle/regulation/verify":
		// Net.ApiBattleRegulationVerifyResponseParam.UniqueParam (Type 13603)
		s.writeGameResponse(w, r, map[string]interface{}{
			"errs":  []int{},
			"derrs": map[string][]int{},
			"UD":    s.Store.BuildFullUD(),
		})

	case "/api/battle/start-matching":
		// Net.ApiBattleStartMatchingResponseParam.UniqueParam (Type 13616)
		s.writeGameResponse(w, r, model.BattleStartMatchingPmp{
			Exp:    int(now + 3600),
			Atk:    "atk_bot_match_001",
			Ehsh:   "ehsh_bot_match_001",
			Ws:     s.resolveWSURL(r),
			Type:   "match",
			Bamin:  900,
			Lmst:   now,
			Mawarn: 0,
			Mast:   now,
			UD:     s.Store.BuildFullUD(),
		})

	case "/api/battle/start-battle":
		// Net.ApiBattleStartBattleResponseParam.UniqueParam (Type 13611)
		exp := int(now + 3600)
		s.writeGameResponse(w, r, model.BattleStartBattlePmp{
			Rmid:   "rm_bot_champion_001",
			Sprule: []model.SpecialRuleSetting{{Typ: 1, Use: 1}},
			Exp:    &exp,
			Atk:    "atk_bot_match_001",
			Ehsh:   "ehsh_bot_match_001",
			Ws:     s.resolveWSURL(r),
			Type:   "battle",
			UD:     s.Store.BuildFullUD(),
		})

	case "/api/battle/resume-matching":
		// Net.ApiBattleResumeMatchingResponseParam.UniqueParam (Type 13607)
		pTeam := s.Store.BuildPlayerBattleTeam()
		bTeam := s.Store.BuildBotBattleTeam(0)
		s.writeGameResponse(w, r, map[string]interface{}{
			"rmid":   "rm_bot_champion_001",
			"bca":    byte(1),
			"rul":    byte(1),
			"eveid":  "season_01",
			"stage":  1,
			"bgm":    1,
			"pkview": 1,
			"pkvtim": 60,
			"sprule": []model.SpecialRuleSetting{{Typ: 1, Use: 1}},
			"team":   []model.ApiTeamParam{pTeam, bTeam},
			"stt":    1,
			"exp":    int64(now + 3600),
			"atk":    "atk_bot_match_001",
			"ehsh":   "ehsh_bot_match_001",
			"ws":     s.resolveWSURL(r),
			"type":   "battle",
			"bamin":  900,
			"rank":   1,
			"UD":     s.Store.BuildFullUD(),
		})

	case "/api/battle/end-battle", "/api/battle/surrender-battle":
		// Net.ApiBattleEndBattleResponseParam.UniqueParam (Type 13583)
		rank := 1
		s.writeGameResponse(w, r, model.BattleEndBattlePmp{
			Bca:   1,
			Rul:   1,
			Stt:   1,
			Pobf:  1500,
			Poaf:  1550,
			Rkbf:  10,
			Rkaf:  10,
			Mrkbf: 10,
			Mrkaf: 10,
			Wicbf: 10,
			Wicaf: 11,
			Bopo:  50,
			Rabf:  1500.0,
			Raaf:  1525.0,
			Et:    1,
			Rwd: []model.ApiBattleEndBattleRewardParam{
				{
					Reason: 1,
					Param:  1,
					Resource: []model.ApiBattleEndUserResourceParam{
						{Type: 1, Id: 1, Num: 500},
					},
				},
			},
			Rank:   &rank,
			Vpbm:   100,
			Rptflg: 0,
			Regag:  1,
			UD:     s.Store.BuildFullUD(),
		})

	case "/api/lobby/create":
		s.writeGameResponse(w, r, map[string]interface{}{
			"exp":  int(now + 3600),
			"atk":  "atk_lobby_001",
			"ehsh": "ehsh_lobby_001",
			"rmid": "rm_lobby_001",
			"ws":   s.resolveWSURL(r),
			"type": "lobby",
			"UD":   s.Store.BuildFullUD(),
		})

	case "/api/lobby/join":
		s.writeGameResponse(w, r, map[string]interface{}{
			"exp":   int(now + 3600),
			"atk":   "atk_lobby_001",
			"ehsh":  "ehsh_lobby_001",
			"ws":    s.resolveWSURL(r),
			"type":  "lobby",
			"rul":   byte(1),
			"regid": "reg_standard_01",
			"UD":    s.Store.BuildFullUD(),
		})

	case "/api/lobby/invited-list":
		s.writeGameResponse(w, r, map[string]interface{}{
			"list": map[string]interface{}{},
			"UD":   s.Store.BuildFullUD(),
		})

	case "/api/competition/top-list":
		s.writeGameResponse(w, r, map[string]interface{}{
			"coms": []interface{}{},
			"UD":   s.Store.BuildFullUD(),
		})

	case "/api/competition/list", "/api/competition/entered-list":
		s.writeGameResponse(w, r, map[string]interface{}{
			"coms":  []interface{}{},
			"total": 0,
			"ldcnt": 0,
			"excnt": 0,
			"ckey":  0,
			"UD":    s.Store.BuildFullUD(),
		})

	case "/api/competition/confirm-result":
		s.writeGameResponse(w, r, map[string]interface{}{
			"coms": map[string]interface{}{},
			"UD":   s.Store.BuildFullUD(),
		})

	case "/api/regular-event/confirm-result":
		s.writeGameResponse(w, r, map[string]interface{}{
			"eves": map[string]interface{}{},
			"UD":   s.Store.BuildFullUD(),
		})

	case "/api/ranking/regular-event/get-my-rank":
		s.writeGameResponse(w, r, map[string]interface{}{
			"ranks": map[string]int{"season_01": 1},
			"UD":    s.Store.BuildFullUD(),
		})

	case "/api/ranking/regular-event/get-my-result":
		s.writeGameResponse(w, r, map[string]interface{}{
			"rslts": map[string]interface{}{},
			"UD":    s.Store.BuildFullUD(),
		})

	case "/api/ranking/competition/get-my-rank":
		rat := float32(1500.0)
		s.writeGameResponse(w, r, map[string]interface{}{
			"rank": 1,
			"rat":  &rat,
			"UD":   s.Store.BuildFullUD(),
		})

	case "/api/friend/list":
		s.writeGameResponse(w, r, map[string]interface{}{
			"mPrfs":   []interface{}{},
			"updated": false,
			"UD":      s.Store.BuildFullUD(),
		})

	case "/api/friend/request":
		s.writeGameResponse(w, r, map[string]interface{}{
			"sFIds": []interface{}{},
			"UD":    s.Store.BuildFullUD(),
		})

	case "/api/message/list":
		s.writeGameResponse(w, r, map[string]interface{}{
			"msgl":   []interface{}{},
			"retyp":  byte(0),
			"lmsgid": "",
			"UD":     s.Store.BuildFullUD(),
		})

	case "/api/information/get-unread":
		s.writeGameResponse(w, r, map[string]interface{}{
			"unrids": []string{},
			"UD":     s.Store.BuildFullUD(),
		})

	case "/api/information/read":
		s.writeGameResponse(w, r, map[string]interface{}{
			"result": true,
			"UD":     s.Store.BuildFullUD(),
		})

	case "/api/ph-transfer/box":
		s.writeGameResponse(w, r, map[string]interface{}{
			"list": st.Pokemons,
			"UD":   s.Store.BuildFullUD(),
		})

	case "/api/ph-transfer/receive":
		s.writeGameResponse(w, r, map[string]interface{}{
			"trrew": []interface{}{},
			"UD":    s.Store.BuildFullUD(),
		})

	case "/api/switch/paid-check":
		s.writeGameResponse(w, r, map[string]interface{}{
			"purc": 1,
			"UD":   s.Store.BuildFullUD(),
		})

	default:
		// Universal handler for all other /api/* and /auth/* routes
		_ = env
		s.writeGameResponse(w, r, map[string]interface{}{
			"UD": s.Store.BuildFullUD(),
		})
	}

	dur := time.Since(start)
	s.recordTraffic(r.Method, path, dur, fmt.Sprintf("200 OK (ENC=%s)", r.Header.Get("X-PKB-ENCRYPT")))
	log.Printf("  └─ Respondido 200 OK em %v", dur)
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
