package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/nicokaka/mypokechamp_server/internal/model"
	"github.com/nicokaka/mypokechamp_server/internal/store"
)

type TrafficEntry struct {
	ID         int64  `json:"id"`
	Time       string `json:"time"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	DurationMs int64  `json:"duration_ms"`
	Note       string `json:"note,omitempty"`
}

func (s *Server) recordTraffic(method, path string, dur time.Duration, note string) {
	s.trafficMu.Lock()
	defer s.trafficMu.Unlock()
	s.trafficSeq++
	entry := TrafficEntry{
		ID:         s.trafficSeq,
		Time:       time.Now().Format("15:04:05"),
		Method:     method,
		Path:       path,
		DurationMs: dur.Milliseconds(),
		Note:       note,
	}
	s.trafficLog = append([]TrafficEntry{entry}, s.trafficLog...)
	if len(s.trafficLog) > 100 {
		s.trafficLog = s.trafficLog[:100]
	}
}

func (s *Server) handleAdminState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(s.Store.GetState())
}

func (s *Server) handleAdminTraffic(w http.ResponseWriter, r *http.Request) {
	s.trafficMu.Lock()
	logs := append([]TrafficEntry(nil), s.trafficLog...)
	s.trafficMu.Unlock()
	if logs == nil {
		logs = []TrafficEntry{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"public_url": s.PublicURL,
		"ws_url":     s.WSURL,
		"logs":       logs,
	})
}

func (s *Server) handleAdminUpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req store.AdminProfileUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	st := s.Store.UpdateProfileAdmin(req)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(st)
}

func (s *Server) handleAdminUpdateAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req model.ApiUserAvatarParam
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.Store.UpdateAvatar(req)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(s.Store.GetState())
}

func (s *Server) handleAdminMaxOut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := s.Store.MaxOutAccount()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(st)
}

func (s *Server) handleAdminUnlockMeta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	added := s.Store.UnlockMetaPack()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"added":  added,
		"state":  s.Store.GetState(),
	})
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
		TeraType uint32   `json:"tera_type"`
		Ball     byte     `json:"ball"`
		EVs      []byte   `json:"evs"`
		IVs      []uint16 `json:"ivs"`
		Moves    []int16  `json:"moves"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Moves) == 0 {
		req.Moves = []int16{85, 89, 53, 182}
	}
	evs := [6]byte{4, 252, 0, 252, 0, 0}
	if len(req.EVs) >= 6 {
		copy(evs[:], req.EVs[:6])
	}
	ivs := []uint16{31, 31, 31, 31, 31, 31}
	if len(req.IVs) >= 6 {
		ivs = req.IVs[:6]
	}
	ball := req.Ball
	if ball == 0 {
		ball = 4
	}
	if req.Nature == 0 {
		req.Nature = 15
	}
	if req.TeraType == 0 {
		req.TeraType = 1
	}
	p := model.ApiUserPokemonParam{
		Ver: 1,
		B1:  req.MonsNo,
		B4:  125000,
		B5:  req.Ability,
		B6:  req.Moves,
		B7:  req.Item,
		B8:  uint16(ball),
		B9:  evs[0],
		Ba:  evs[1],
		Bb:  evs[2],
		Bc:  evs[3],
		Bd:  evs[4],
		Be:  evs[5],
		Bf:  ivs,
		Bj:  req.Nature,
		Bk:  req.Nature,
		Bl:  req.Shiny,
		Bm:  req.TeraType,
		Bn:  2,
		Bo:  req.Nickname,
	}
	s.Store.AddOrUpdatePokemon(req.UID, p)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "state": s.Store.GetState()})
}

func (s *Server) handleAdminDeletePokemon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		UID  string   `json:"uid"`
		UIDs []string `json:"uids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	uids := req.UIDs
	if req.UID != "" {
		uids = append(uids, req.UID)
	}
	s.Store.DeletePokemons(uids)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "state": s.Store.GetState()})
}

func (s *Server) handleAdminSaveTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req model.ApiUserTeamParam
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	st := s.Store.GetState()
	itms := make([]model.ApiUserTeamItemParam, 0, len(req.Mem))
	for _, uid := range req.Mem {
		if pk, ok := st.Pokemons[uid]; ok {
			itms = append(itms, model.ApiUserTeamItemParam{P: uid, I: int(pk.B7)})
		}
	}
	req.Itms = itms
	if req.Lng == 0 {
		req.Lng = 2
	}
	s.Store.UpdateTeam(req.Tid, req)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "state": s.Store.GetState()})
}

func (s *Server) handleAdminDeleteTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Tid int `json:"tid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.Store.DeleteTeam(req.Tid)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "state": s.Store.GetState()})
}

func (s *Server) handleAdminSaveBot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req model.MdBotMergeParam
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	st := s.Store.SaveBot(req)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "state": st})
}

func (s *Server) handleAdminDeleteBot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	st := s.Store.DeleteBot(req.ID)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "state": st})
}

func (s *Server) handleAdminReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := s.Store.ResetToDefault()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "state": st})
}

func (s *Server) handleAdminUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminHTML))
}

const adminHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>MyPokeChamp Private Server — Painel de Controle</title>
<style>
  :root {
    --bg: #0b0f19;
    --sidebar: #111827;
    --panel: #161f33;
    --panel-hover: #1e293b;
    --border: #26334d;
    --text: #f8fafc;
    --muted: #94a3b8;
    --accent: #38bdf8;
    --accent-2: #6366f1;
    --emerald: #10b981;
    --amber: #f59e0b;
    --danger: #ef4444;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    background: var(--bg);
    color: var(--text);
    font-family: 'Inter', system-ui, -apple-system, sans-serif;
    display: flex;
    min-height: 100vh;
  }
  /* Sidebar */
  .sidebar {
    width: 260px;
    background: var(--sidebar);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
  }
  .brand {
    padding: 20px 18px;
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .brand-logo {
    width: 40px;
    height: 40px;
    border-radius: 10px;
    background: linear-gradient(135deg, #38bdf8, #6366f1);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    font-weight: 800;
    box-shadow: 0 4px 14px rgba(56, 189, 248, 0.3);
  }
  .brand-text strong { display: block; font-size: 15px; letter-spacing: -0.01em; }
  .brand-text small { color: var(--muted); font-size: 11px; }
  .nav-list {
    padding: 14px 10px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    flex: 1;
  }
  .nav-btn {
    width: 100%;
    text-align: left;
    background: transparent;
    color: var(--muted);
    border: 1px solid transparent;
    padding: 11px 14px;
    border-radius: 10px;
    font-size: 13.5px;
    font-weight: 600;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: space-between;
    transition: all 0.15s ease;
  }
  .nav-btn:hover { background: rgba(255,255,255,0.04); color: var(--text); }
  .nav-btn.active {
    background: rgba(56, 189, 248, 0.12);
    color: var(--accent);
    border-color: rgba(56, 189, 248, 0.3);
  }
  .badge {
    font-size: 11px;
    padding: 2px 7px;
    border-radius: 999px;
    background: rgba(255,255,255,0.08);
    color: var(--text);
  }
  .sidebar-footer {
    padding: 14px 16px;
    border-top: 1px solid var(--border);
    font-size: 12px;
    color: var(--muted);
  }
  .status-pill {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    color: var(--emerald);
    font-weight: 700;
    margin-bottom: 4px;
  }
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--emerald);
    box-shadow: 0 0 8px var(--emerald);
  }

  /* Main Area */
  .main {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .topbar {
    padding: 16px 28px;
    background: rgba(17, 24, 39, 0.85);
    backdrop-filter: blur(8px);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    flex-wrap: wrap;
    position: sticky;
    top: 0;
    z-index: 20;
  }
  .trainer-summary {
    display: flex;
    align-items: center;
    gap: 14px;
  }
  .trainer-avatar {
    width: 46px;
    height: 46px;
    border-radius: 12px;
    background: linear-gradient(135deg, rgba(16,185,129,0.2), rgba(56,189,248,0.2));
    border: 1px solid rgba(56,189,248,0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    font-weight: 800;
    color: var(--accent);
  }
  .trainer-meta h2 {
    margin: 0;
    font-size: 18px;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .trainer-meta p {
    margin: 2px 0 0;
    font-size: 12px;
    color: var(--muted);
  }
  .top-actions {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }
  .btn {
    padding: 9px 15px;
    border-radius: 8px;
    border: 1px solid var(--border);
    background: var(--panel);
    color: var(--text);
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    transition: all 0.15s ease;
  }
  .btn:hover { background: var(--panel-hover); border-color: #3b4f75; }
  .btn-primary {
    background: linear-gradient(135deg, #0284c7, #2563eb);
    border-color: #38bdf8;
    color: #fff;
  }
  .btn-primary:hover { filter: brightness(1.1); }
  .btn-emerald {
    background: linear-gradient(135deg, #059669, #10b981);
    border-color: #34d399;
    color: #fff;
  }
  .btn-amber {
    background: linear-gradient(135deg, #d97706, #f59e0b);
    border-color: #fbbf24;
    color: #fff;
  }
  .btn-danger {
    background: rgba(239, 68, 68, 0.15);
    border-color: rgba(239, 68, 68, 0.4);
    color: #fca5a5;
  }
  .btn-danger:hover { background: rgba(239, 68, 68, 0.28); }

  .content {
    padding: 24px 28px 60px;
    max-width: 1360px;
    width: 100%;
    margin: 0 auto;
  }
  .toast {
    display: none;
    margin-bottom: 18px;
    padding: 12px 16px;
    border-radius: 10px;
    background: rgba(16, 185, 129, 0.15);
    border: 1px solid rgba(16, 185, 129, 0.45);
    color: #6ee7b7;
    font-weight: 600;
    font-size: 13.5px;
    align-items: center;
    justify-content: space-between;
  }
  .toast.show { display: flex; }

  .tab-pane { display: none; }
  .tab-pane.active { display: block; }

  .card {
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 20px;
    margin-bottom: 20px;
  }
  .card h3 {
    margin: 0 0 6px;
    font-size: 16px;
    color: var(--text);
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .card p.sub {
    margin: 0 0 16px;
    font-size: 12.5px;
    color: var(--muted);
  }
  .form-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
    gap: 14px;
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }
  .field label {
    font-size: 11.5px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--muted);
  }
  .field input, .field select {
    padding: 9px 11px;
    border-radius: 8px;
    border: 1px solid var(--border);
    background: #0b0f19;
    color: var(--text);
    font-size: 13.5px;
    outline: none;
  }
  .field input:focus, .field select:focus {
    border-color: var(--accent);
  }

  /* Stat KPIs */
  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
    gap: 14px;
    margin-bottom: 20px;
  }
  .kpi {
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 16px;
  }
  .kpi small { color: var(--muted); font-size: 11.5px; font-weight: 600; text-transform: uppercase; }
  .kpi strong { display: block; font-size: 22px; margin-top: 4px; color: var(--accent); }

  /* Pokemon Grid */
  .preset-bar {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
    margin-bottom: 14px;
  }
  .preset-chip {
    padding: 5px 10px;
    border-radius: 999px;
    font-size: 12px;
    font-weight: 600;
    background: rgba(56, 189, 248, 0.1);
    border: 1px solid rgba(56, 189, 248, 0.28);
    color: #bae6fd;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 5px;
  }
  .preset-chip:hover { background: rgba(56, 189, 248, 0.22); }

  .poke-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(255px, 1fr));
    gap: 14px;
  }
  .poke-card {
    background: #0d1322;
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    position: relative;
    transition: transform 0.12s ease, border-color 0.12s ease;
  }
  .poke-card:hover { transform: translateY(-2px); border-color: #3b4f75; }
  .poke-card.shiny { border-color: rgba(250, 204, 21, 0.55); box-shadow: 0 0 18px rgba(250, 204, 21, 0.08); }
  .poke-top {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .poke-sprite {
    width: 64px;
    height: 64px;
    border-radius: 10px;
    background: rgba(255,255,255,0.03);
    border: 1px solid rgba(255,255,255,0.06);
    object-fit: contain;
  }
  .poke-title strong { font-size: 15px; display: block; }
  .poke-title small { color: var(--muted); font-size: 11.5px; }
  .poke-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 5px;
    font-size: 11px;
  }
  .tag {
    padding: 2px 7px;
    border-radius: 5px;
    background: rgba(255,255,255,0.06);
    color: #cbd5e1;
  }
  .tag-shiny { background: rgba(250, 204, 21, 0.18); color: #fde047; font-weight: 700; }
  .poke-actions {
    display: flex;
    gap: 6px;
    margin-top: auto;
  }
  .poke-actions button { flex: 1; padding: 6px 10px; font-size: 12px; }

  /* Traffic Table */
  table.traffic-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }
  table.traffic-table th, table.traffic-table td {
    text-align: left;
    padding: 10px 12px;
    border-bottom: 1px solid var(--border);
  }
  table.traffic-table th { color: var(--muted); font-size: 11.5px; text-transform: uppercase; }
  .method-badge {
    padding: 2px 7px;
    border-radius: 5px;
    font-size: 11px;
    font-weight: 700;
    background: rgba(56, 189, 248, 0.15);
    color: var(--accent);
  }

  @media (max-width: 820px) {
    body { flex-direction: column; }
    .sidebar { width: 100%; border-right: none; border-bottom: 1px solid var(--border); }
    .nav-list { flex-direction: row; overflow-x: auto; }
    .nav-btn { white-space: nowrap; width: auto; }
  }
</style>
</head>
<body>
  <aside class="sidebar">
    <div class="brand">
      <div class="brand-logo">🏆</div>
      <div class="brand-text">
        <strong>MyPokeChamp Studio</strong>
        <small>Private Server v1.2.2-3859</small>
      </div>
    </div>
    <nav class="nav-list">
      <button class="nav-btn active" onclick="switchTab('profile', this)">
        <span>👤 Perfil & Ranking</span>
        <span class="badge" id="nav-rank">Rank 10</span>
      </button>
      <button class="nav-btn" onclick="switchTab('box', this)">
        <span>📦 Box Pokémon (Showdown)</span>
        <span class="badge" id="nav-pokes">0</span>
      </button>
      <button class="nav-btn" onclick="switchTab('teams', this)">
        <span>⚔️ Times de Batalha</span>
        <span class="badge" id="nav-teams">0</span>
      </button>
      <button class="nav-btn" onclick="switchTab('bots', this)">
        <span>🤖 Bots & Matchmaking</span>
        <span class="badge" id="nav-bots">0</span>
      </button>
      <button class="nav-btn" onclick="switchTab('traffic', this)">
        <span>📡 Tráfego em Tempo Real</span>
        <span class="badge" id="nav-traffic">Live</span>
      </button>
    </nav>
    <div class="sidebar-footer">
      <div class="status-pill"><span class="dot"></span> Servidor Privado Online</div>
      <div id="server-url-label">Oracle Cloud 24/7 • Porta 9443</div>
    </div>
  </aside>

  <div class="main">
    <header class="topbar">
      <div class="trainer-summary">
        <div class="trainer-avatar" id="top-initial">N</div>
        <div class="trainer-meta">
          <h2>
            <span id="top-name">Nicolas</span>
            <span class="tag" style="background:rgba(16,185,129,0.2);color:#34d399;border:1px solid rgba(16,185,129,0.4)">Conta Ativa</span>
          </h2>
          <p>Trainer ID: <strong id="top-trnid">CHAMP-2026-0001</strong> • Friend Code: <strong id="top-fri">1234-5678-9012</strong> • UdVer: <strong id="top-udver">1</strong></p>
        </div>
      </div>
      <div class="top-actions">
        <button class="btn btn-amber" onclick="unlockMetaPack()">✨ Liberar Pack Lendários & Meta VGC</button>
        <button class="btn btn-emerald" onclick="maxOutAccount()">⚡ Max Out (9.999.999 VP + Master Ball)</button>
        <button class="btn" onclick="loadAll()">🔄 Atualizar</button>
      </div>
    </header>

    <div class="content">
      <div id="toast" class="toast">
        <span id="toast-msg">Alterações salvas com sucesso!</span>
        <button class="btn" style="padding:3px 8px;font-size:11px" onclick="document.getElementById('toast').classList.remove('show')">Fechar</button>
      </div>

      <div class="kpi-grid">
        <div class="kpi">
          <small>Victory Points (VP)</small>
          <strong id="kpi-vp">999.999</strong>
        </div>
        <div class="kpi">
          <small>Pokémon na Box</small>
          <strong id="kpi-box">12 / 300</strong>
        </div>
        <div class="kpi">
          <small>Elo Ranked Singles</small>
          <strong id="kpi-single">1500 (Rank 10)</strong>
        </div>
        <div class="kpi">
          <small>Elo Ranked Doubles</small>
          <strong id="kpi-double">1500 (Rank 10)</strong>
        </div>
      </div>

      <!-- TAB 1: PERFIL & RANKING -->
      <section id="tab-profile" class="tab-pane active">
        <div class="card">
          <h3>👤 Identidade do Treinador & Recursos</h3>
          <p class="sub">Altere seu nome no jogo, código de treinador, moedas (Victory Points), capacidade da Box e estatísticas da temporada.</p>
          <div class="form-grid">
            <div class="field">
              <label>Nome do Treinador (nam)</label>
              <input id="prf-nam" type="text" placeholder="Ex: Nicolas">
            </div>
            <div class="field">
              <label>Trainer ID (trn_id)</label>
              <input id="prf-trnid" type="text" placeholder="CHAMP-2026-0001">
            </div>
            <div class="field">
              <label>Friend Code (fri)</label>
              <input id="prf-fri" type="text" placeholder="1234-5678-9012">
            </div>
            <div class="field">
              <label>Victory Points (VP / Moedas)</label>
              <input id="prf-vp" type="number" value="999999">
            </div>
            <div class="field">
              <label>Capacidade da Box (pbsiz)</label>
              <input id="prf-pbsiz" type="number" value="300">
            </div>
            <div class="field">
              <label>Season Points (Passe)</label>
              <input id="prf-sp" type="number" value="1000">
            </div>
            <div class="field">
              <label>Título (trtit)</label>
              <input id="prf-trtit" type="text" value="1">
            </div>
            <div class="field">
              <label>Ícone de Perfil (trico)</label>
              <input id="prf-trico" type="text" value="1">
            </div>
            <div class="field">
              <label>Idioma (lng)</label>
              <select id="prf-lng">
                <option value="2">Inglês / Global (2)</option>
                <option value="1">Japonês (1)</option>
                <option value="7">Espanhol (7)</option>
                <option value="8">Português (8)</option>
              </select>
            </div>
            <div class="field">
              <label>País / Região (cou)</label>
              <input id="prf-cou" type="number" value="31">
            </div>
          </div>
        </div>

        <div class="card">
          <h3>🏆 Ranking Competitivo (Singles & Doubles)</h3>
          <p class="sub">Configure seu Tier (Rank 10 = Master Ball Tier), Rating (Elo) e número de vitórias exibidos no cartão de batalha.</p>
          <div class="form-grid">
            <div class="field">
              <label>Singles Rank (1 a 10)</label>
              <input id="prf-rk-s" type="number" min="1" max="10" value="10">
            </div>
            <div class="field">
              <label>Singles Rating (Elo)</label>
              <input id="prf-rat-s" type="number" step="1" value="1500">
            </div>
            <div class="field">
              <label>Singles Vitórias</label>
              <input id="prf-win-s" type="number" value="10">
            </div>
            <div class="field">
              <label>Singles Derrotas</label>
              <input id="prf-los-s" type="number" value="0">
            </div>
            <div class="field">
              <label>Doubles Rank (1 a 10)</label>
              <input id="prf-rk-d" type="number" min="1" max="10" value="10">
            </div>
            <div class="field">
              <label>Doubles Rating (Elo)</label>
              <input id="prf-rat-d" type="number" step="1" value="1500">
            </div>
            <div class="field">
              <label>Doubles Vitórias</label>
              <input id="prf-win-d" type="number" value="10">
            </div>
            <div class="field">
              <label>Doubles Derrotas</label>
              <input id="prf-los-d" type="number" value="0">
            </div>
          </div>
          <div style="margin-top:18px;display:flex;gap:10px;flex-wrap:wrap">
            <button class="btn btn-primary" onclick="saveProfile()">💾 Salvar Perfil & Ranking</button>
            <button class="btn btn-danger" onclick="resetAccount()">⚠️ Restaurar Conta Padrão</button>
          </div>
        </div>
      </section>

      <!-- TAB 2: BOX POKEMON (SHOWDOWN EDITOR) -->
      <section id="tab-box" class="tab-pane">
        <div class="card">
          <h3>
            <span>⚡ Pokémon Showdown Builder — Criar ou Editar Pokémon na Box</span>
            <button class="btn" style="font-size:12px;padding:5px 10px" onclick="clearPokeForm()">Limpar Formulário</button>
          </h3>
          <p class="sub">Clique em um Preset competitivo abaixo ou preencha qualquer Dex ID (1 a 1025) com IVs 6x31, EVs, Natureza, Tera Type e Moves.</p>

          <div class="preset-bar">
            <span class="preset-chip" onclick="fillPreset(6,'Charizard',66,660,15,2,1,[4,0,0,252,252,0],[53,403,76,182])">🔥 Charizard Shiny</span>
            <span class="preset-chip" onclick="fillPreset(150,'Mewtwo',46,270,15,14,1,[4,0,0,252,252,0],[540,94,58,182])">🔮 Mewtwo Shiny</span>
            <span class="preset-chip" onclick="fillPreset(384,'Rayquaza',76,275,13,2,1,[4,252,0,252,0,0],[620,245,349,182])">🐉 Rayquaza Shiny</span>
            <span class="preset-chip" onclick="fillPreset(445,'Garchomp',134,270,13,5,1,[4,252,0,252,0,0],[89,337,442,182])">🦈 Garchomp</span>
            <span class="preset-chip" onclick="fillPreset(1008,'Miraidon',289,297,15,4,0,[4,0,0,252,252,0],[879,434,521,605])">⚡ Miraidon</span>
            <span class="preset-chip" onclick="fillPreset(1007,'Koraidon',288,1699,13,10,0,[4,252,0,252,0,0],[878,394,888,182])">☀️ Koraidon</span>
            <span class="preset-chip" onclick="fillPreset(987,'Flutter Mane',281,1880,15,18,1,[4,0,0,252,252,0],[585,247,605,182])">👻 Flutter Mane</span>
            <span class="preset-chip" onclick="fillPreset(888,'Zacian',234,1103,13,18,1,[4,252,0,252,0,0],[781,583,370,182])">🗡️ Zacian Shiny</span>
            <span class="preset-chip" onclick="fillPreset(892,'Urshifu',260,275,13,11,0,[4,252,0,252,0,0],[818,370,453,197])">🥋 Urshifu</span>
            <span class="preset-chip" onclick="fillPreset(727,'Incineroar',22,1120,8,8,1,[252,4,156,0,0,96],[252,394,282,575])">🐯 Incineroar</span>
          </div>

          <div class="form-grid">
            <div class="field">
              <label>UID na Box (vazio = novo)</label>
              <input id="pk-uid" type="text" placeholder="Ex: pk_001 ou deixe vazio">
            </div>
            <div class="field">
              <label>National Dex # (b1)</label>
              <input id="pk-monsno" type="number" value="150">
            </div>
            <div class="field">
              <label>Nome / Nickname (bo)</label>
              <input id="pk-nick" type="text" value="Mewtwo">
            </div>
            <div class="field">
              <label>Shiny (bl)</label>
              <select id="pk-shiny">
                <option value="1">✨ Sim (Shiny)</option>
                <option value="0">Normal</option>
              </select>
            </div>
            <div class="field">
              <label>Ability ID (b5)</label>
              <input id="pk-ability" type="number" value="46">
            </div>
            <div class="field">
              <label>Held Item ID (b7)</label>
              <input id="pk-item" type="number" value="270" title="270=Life Orb, 275=Focus Sash, 234=Leftovers, 640=Assault Vest">
            </div>
            <div class="field">
              <label>Natureza (bj)</label>
              <select id="pk-nature">
                <option value="15">Timid (+Spe, -Atk) [15]</option>
                <option value="13">Jolly (+Spe, -SpA) [13]</option>
                <option value="3">Adamant (+Atk, -SpA) [3]</option>
                <option value="17">Modest (+SpA, -Atk) [17]</option>
                <option value="8">Impish (+Def, -SpA) [8]</option>
                <option value="22">Careful (+SpD, -SpA) [22]</option>
                <option value="0">Hardy (Neutra) [0]</option>
              </select>
            </div>
            <div class="field">
              <label>Tera Type (bm)</label>
              <select id="pk-tera">
                <option value="1">Normal (1)</option>
                <option value="2">Lutador / Voador (2)</option>
                <option value="3">Água / Escuro (3)</option>
                <option value="4">Elétrico (4)</option>
                <option value="5">Terra (5)</option>
                <option value="8">Fantasma (8)</option>
                <option value="9">Aço (9)</option>
                <option value="10">Fogo (10)</option>
                <option value="11">Água (11)</option>
                <option value="14">Psíquico (14)</option>
                <option value="18">Fada (18)</option>
                <option value="99">Stellar (99)</option>
              </select>
            </div>
            <div class="field">
              <label>EVs (HP,Atk,Def,Spe,SpA,SpD)</label>
              <input id="pk-evs" type="text" value="4,0,0,252,252,0">
            </div>
            <div class="field">
              <label>4 Moves IDs (b6)</label>
              <input id="pk-moves" type="text" value="540,94,58,182">
            </div>
          </div>
          <div style="margin-top:16px">
            <button class="btn btn-primary" onclick="savePokemon()">➕ Salvar / Atualizar Pokémon na Box</button>
          </div>
        </div>

        <div class="card">
          <h3>📦 Pokémon na Box (<span id="box-count">0</span>)</h3>
          <p class="sub">Clique em "Editar" em qualquer Pokémon para carregá-lo no editor acima ou "Excluir" para removê-lo.</p>
          <div id="poke-list" class="poke-grid"></div>
        </div>
      </section>

      <!-- TAB 3: TIMES DE BATALHA -->
      <section id="tab-teams" class="tab-pane">
        <div class="card">
          <h3>⚔️ Editor de Time Competitivo (6 Slots)</h3>
          <p class="sub">Configure os membros do seu time principal de Batalha Ranqueada / Casual.</p>
          <div class="form-grid">
            <div class="field">
              <label>Slot do Time (tid)</label>
              <input id="tm-tid" type="number" min="1" max="10" value="1">
            </div>
            <div class="field">
              <label>Nome do Time (nam)</label>
              <input id="tm-nam" type="text" value="Champions Main">
            </div>
            <div class="field">
              <label>Time Principal Ativo? (cuflg)</label>
              <select id="tm-active">
                <option value="1">Sim (Ativo em Singles & Doubles)</option>
                <option value="0">Secundário</option>
              </select>
            </div>
            <div class="field" style="grid-column: span 2">
              <label>6 UIDs da Box (separados por vírgula)</label>
              <input id="tm-mem" type="text" value="pk_001,pk_002,pk_003,pk_004,pk_005,pk_006">
            </div>
          </div>
          <div style="margin-top:16px">
            <button class="btn btn-primary" onclick="saveTeam()">💾 Salvar Time de Batalha</button>
          </div>
        </div>

        <div class="card">
          <h3>📋 Times Cadastrados</h3>
          <div id="team-list" class="poke-grid"></div>
        </div>
      </section>

      <!-- TAB 4: BOTS & MATCHMAKING -->
      <section id="tab-bots" class="tab-pane">
        <div class="card">
          <h3>🤖 Configurar Oponente Sintético (Bot de Matchmaking)</h3>
          <p class="sub">Adicione ou edite Campeões que enfrentam você instantaneamente na fila de batalha (WebSocket2 & HTTP).</p>
          <div class="form-grid">
            <div class="field">
              <label>Bot ID</label>
              <input id="bot-id" type="number" value="1001">
            </div>
            <div class="field">
              <label>Nome do Campeão / Bot</label>
              <input id="bot-name" type="text" value="Champion Cynthia">
            </div>
            <div class="field">
              <label>Rating / Elo (ra)</label>
              <input id="bot-ra" type="number" value="1850">
            </div>
            <div class="field" style="grid-column: span 2">
              <label>6 Dex IDs do Time do Bot (separados por vírgula)</label>
              <input id="bot-mons" type="text" value="445,448,350,468,442,407">
            </div>
          </div>
          <div style="margin-top:16px">
            <button class="btn btn-primary" onclick="saveBot()">💾 Salvar Bot</button>
          </div>
        </div>

        <div class="card">
          <h3>🏟️ Bots Ativos na Arena</h3>
          <div id="bot-list" class="poke-grid"></div>
        </div>
      </section>

      <!-- TAB 5: TRAFEGO EM TEMPO REAL -->
      <section id="tab-traffic" class="tab-pane">
        <div class="card">
          <h3>
            <span>📡 Monitor de Tráfego HTTP / WebSocket2</span>
            <button class="btn" style="font-size:12px;padding:5px 10px" onclick="loadTraffic()">🔄 Atualizar Logs</button>
          </h3>
          <p class="sub">Últimas requisições recebidas pelo servidor privado a partir do APK Pokémon Champions.</p>
          <table class="traffic-table">
            <thead>
              <tr>
                <th>Horário</th>
                <th>Método</th>
                <th>Endpoint</th>
                <th>Tempo</th>
                <th>Detalhes</th>
              </tr>
            </thead>
            <tbody id="traffic-body">
              <tr><td colspan="5" style="color:var(--muted)">Carregando tráfego...</td></tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </div>

<script>
let currentState = null;

function showToast(msg) {
  const t = document.getElementById('toast');
  document.getElementById('toast-msg').innerText = msg;
  t.classList.add('show');
  setTimeout(() => t.classList.remove('show'), 4000);
}

function switchTab(tabId, btn) {
  document.querySelectorAll('.tab-pane').forEach(el => el.classList.remove('active'));
  document.querySelectorAll('.nav-btn').forEach(el => el.classList.remove('active'));
  document.getElementById('tab-' + tabId).classList.add('active');
  if (btn) btn.classList.add('active');
  if (tabId === 'traffic') loadTraffic();
}

function spriteUrl(monsNo, shiny) {
  if (shiny) {
    return 'https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/other/home/shiny/' + monsNo + '.png';
  }
  return 'https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/other/home/' + monsNo + '.png';
}

function renderState(st) {
  currentState = st;
  const prf = st.profile || {};
  const main = st.main || {};
  const pokesMap = st.pokemons || {};
  const pokes = Object.values(pokesMap).sort((a,b) => (a.a1 || '').localeCompare(b.a1 || ''));
  const teams = Object.values(st.teams || {});
  const bots = st.bots || [];

  // Header & KPIs
  document.getElementById('top-name').innerText = prf.nam || 'Nicolas';
  document.getElementById('top-initial').innerText = (prf.nam || 'N').slice(0,1).toUpperCase();
  document.getElementById('top-trnid').innerText = prf.trn_id || 'CHAMP-2026-0001';
  document.getElementById('top-fri').innerText = prf.fri || '1234-5678-9012';
  document.getElementById('top-udver').innerText = st.ud_ver || 1;

  document.getElementById('nav-rank').innerText = 'Rank ' + (st.rank_single || 10);
  document.getElementById('nav-pokes').innerText = pokes.length;
  document.getElementById('nav-teams').innerText = teams.length;
  document.getElementById('nav-bots').innerText = bots.length;

  document.getElementById('kpi-vp').innerText = (main.vp || 0).toLocaleString('pt-BR');
  document.getElementById('kpi-box').innerText = pokes.length + ' / ' + (main.pbsiz || 300);
  document.getElementById('kpi-single').innerText = (st.rating_single || 1500) + ' (Rank ' + (st.rank_single || 10) + ')';
  document.getElementById('kpi-double').innerText = (st.rating_double || 1500) + ' (Rank ' + (st.rank_double || 10) + ')';

  // Profile inputs
  document.getElementById('prf-nam').value = prf.nam || '';
  document.getElementById('prf-trnid').value = prf.trn_id || '';
  document.getElementById('prf-fri').value = prf.fri || '';
  document.getElementById('prf-vp').value = main.vp || 999999;
  document.getElementById('prf-pbsiz').value = main.pbsiz || 300;
  document.getElementById('prf-sp').value = st.season_points || 1000;
  document.getElementById('prf-trtit').value = prf.trtit || '1';
  document.getElementById('prf-trico').value = prf.trico || '1';
  document.getElementById('prf-lng').value = String(prf.lng || 2);
  document.getElementById('prf-cou').value = prf.cou || 31;

  document.getElementById('prf-rk-s').value = st.rank_single || 10;
  document.getElementById('prf-rat-s').value = st.rating_single || 1500;
  document.getElementById('prf-win-s').value = st.wins_single ?? 10;
  document.getElementById('prf-los-s').value = st.losses_single ?? 0;

  document.getElementById('prf-rk-d').value = st.rank_double || 10;
  document.getElementById('prf-rat-d').value = st.rating_double || 1500;
  document.getElementById('prf-win-d').value = st.wins_double ?? 10;
  document.getElementById('prf-los-d').value = st.losses_double ?? 0;

  // Box Pokemon Grid
  document.getElementById('box-count').innerText = pokes.length;
  document.getElementById('poke-list').innerHTML = pokes.map(p => {
    const evs = [p.b9||0, p.ba||0, p.bb||0, p.bc||0, p.bd||0, p.be||0].join('/');
    const moves = (p.b6 || []).join(', ');
    return '<div class="poke-card ' + (p.bl ? 'shiny' : '') + '">' +
      '<div class="poke-top">' +
        '<img class="poke-sprite" src="' + spriteUrl(p.b1, p.bl) + '" onerror="this.src=\'https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/items/poke-ball.png\'" alt="">' +
        '<div class="poke-title">' +
          '<strong>#' + p.b1 + ' ' + (p.bo || 'Pokémon') + (p.bl ? ' ✨' : '') + '</strong>' +
          '<small>UID: <code>' + p.a1 + '</code> • OT: ' + (p.bx || prf.nam) + '</small>' +
        '</div>' +
      '</div>' +
      '<div class="poke-tags">' +
        (p.bl ? '<span class="tag tag-shiny">SHINY</span>' : '') +
        '<span class="tag">Ability: ' + p.b5 + '</span>' +
        '<span class="tag">Item: ' + p.b7 + '</span>' +
        '<span class="tag">Tera: ' + p.bm + '</span>' +
        '<span class="tag">EVs: ' + evs + '</span>' +
        '<span class="tag">Moves: ' + moves + '</span>' +
      '</div>' +
      '<div class="poke-actions">' +
        '<button class="btn" onclick="editPokemon(\'' + p.a1 + '\')">✏️ Editar</button>' +
        '<button class="btn btn-danger" onclick="deletePokemon(\'' + p.a1 + '\')">🗑️ Excluir</button>' +
      '</div>' +
    '</div>';
  }).join('');

  // Teams Grid
  document.getElementById('team-list').innerHTML = teams.map(t => {
    const members = (t.mem || []).map(uid => {
      const pk = pokesMap[uid];
      return pk ? ('#' + pk.b1 + ' ' + pk.bo) : uid;
    }).join(' • ');
    return '<div class="poke-card">' +
      '<div class="poke-title">' +
        '<strong>Time #' + t.tid + ': ' + t.nam + (t.cuflg ? ' 🏆' : '') + '</strong>' +
        '<small>Membros (' + (t.mem||[]).length + '): ' + members + '</small>' +
      '</div>' +
      '<div class="poke-actions">' +
        '<button class="btn" onclick="editTeam(' + t.tid + ')">✏️ Carregar</button>' +
        '<button class="btn btn-danger" onclick="deleteTeam(' + t.tid + ')">🗑️ Excluir</button>' +
      '</div>' +
    '</div>';
  }).join('');

  // Bots Grid
  document.getElementById('bot-list').innerHTML = bots.map(b => {
    return '<div class="poke-card">' +
      '<div class="poke-title">' +
        '<strong>🤖 ' + b.trainer_name + ' (ID ' + b.id + ')</strong>' +
        '<small>Elo Rating: <b>' + b.ra + '</b> • Dex Time: ' + (b.mons || []).join(', ') + '</small>' +
      '</div>' +
      '<div class="poke-actions">' +
        '<button class="btn" onclick="editBot(' + b.id + ')">✏️ Editar</button>' +
        '<button class="btn btn-danger" onclick="deleteBot(' + b.id + ')">🗑️ Remover</button>' +
      '</div>' +
    '</div>';
  }).join('');
}

async function loadAll() {
  const res = await fetch('/admin/api/state');
  const st = await res.json();
  renderState(st);
}

async function saveProfile() {
  const body = {
    nam: document.getElementById('prf-nam').value,
    trn_id: document.getElementById('prf-trnid').value,
    fri: document.getElementById('prf-fri').value,
    vp: parseInt(document.getElementById('prf-vp').value) || 0,
    pbsiz: parseInt(document.getElementById('prf-pbsiz').value) || 300,
    season_points: parseInt(document.getElementById('prf-sp').value) || 1000,
    trtit: document.getElementById('prf-trtit').value,
    trico: document.getElementById('prf-trico').value,
    lng: parseInt(document.getElementById('prf-lng').value) || 2,
    cou: parseInt(document.getElementById('prf-cou').value) || 31,
    rank_single: parseInt(document.getElementById('prf-rk-s').value) || 10,
    rating_single: parseFloat(document.getElementById('prf-rat-s').value) || 1500,
    wins_single: parseInt(document.getElementById('prf-win-s').value) || 0,
    losses_single: parseInt(document.getElementById('prf-los-s').value) || 0,
    rank_double: parseInt(document.getElementById('prf-rk-d').value) || 10,
    rating_double: parseFloat(document.getElementById('prf-rat-d').value) || 1500,
    wins_double: parseInt(document.getElementById('prf-win-d').value) || 0,
    losses_double: parseInt(document.getElementById('prf-los-d').value) || 0
  };
  const res = await fetch('/admin/api/profile', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(body)
  });
  renderState(await res.json());
  showToast('✓ Perfil, Moedas (VP) e Ranking atualizados com sucesso!');
}

async function maxOutAccount() {
  const res = await fetch('/admin/api/maxout', { method: 'POST' });
  renderState(await res.json());
  showToast('⚡ Conta maximizada: 9.999.999 VP, Box 900 slots e Master Ball Tier 2100 Elo!');
}

async function unlockMetaPack() {
  const res = await fetch('/admin/api/unlock-meta', { method: 'POST' });
  const data = await res.json();
  renderState(data.state);
  showToast('✨ Pack Meta VGC/Singles liberado! (' + data.added + ' novos Pokémon competitivos adicionados à Box)');
}

function fillPreset(monsNo, nick, ability, item, nature, tera, shiny, evs, moves) {
  document.getElementById('pk-uid').value = '';
  document.getElementById('pk-monsno').value = monsNo;
  document.getElementById('pk-nick').value = nick;
  document.getElementById('pk-ability').value = ability;
  document.getElementById('pk-item').value = item;
  document.getElementById('pk-nature').value = String(nature);
  document.getElementById('pk-tera').value = String(tera);
  document.getElementById('pk-shiny').value = String(shiny);
  document.getElementById('pk-evs').value = evs.join(',');
  document.getElementById('pk-moves').value = moves.join(',');
}

function clearPokeForm() {
  document.getElementById('pk-uid').value = '';
  document.getElementById('pk-nick').value = '';
}

function editPokemon(uid) {
  const p = (currentState.pokemons || {})[uid];
  if (!p) return;
  document.getElementById('pk-uid').value = p.a1;
  document.getElementById('pk-monsno').value = p.b1;
  document.getElementById('pk-nick').value = p.bo || '';
  document.getElementById('pk-ability').value = p.b5 || 1;
  document.getElementById('pk-item').value = p.b7 || 270;
  document.getElementById('pk-nature').value = String(p.bj ?? 15);
  document.getElementById('pk-tera').value = String(p.bm || 1);
  document.getElementById('pk-shiny').value = String(p.bl || 0);
  document.getElementById('pk-evs').value = [p.b9||0, p.ba||0, p.bb||0, p.bc||0, p.bd||0, p.be||0].join(',');
  document.getElementById('pk-moves').value = (p.b6 || []).join(',');
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

async function savePokemon() {
  const evs = document.getElementById('pk-evs').value.split(',').map(x => parseInt(x.trim()) || 0);
  const moves = document.getElementById('pk-moves').value.split(',').map(x => parseInt(x.trim()) || 0).filter(x => x > 0);
  const body = {
    uid: document.getElementById('pk-uid').value.trim(),
    mons_no: parseInt(document.getElementById('pk-monsno').value) || 6,
    nickname: document.getElementById('pk-nick').value.trim() || 'Pokemon',
    ability: parseInt(document.getElementById('pk-ability').value) || 1,
    item: parseInt(document.getElementById('pk-item').value) || 270,
    nature: parseInt(document.getElementById('pk-nature').value) || 15,
    tera_type: parseInt(document.getElementById('pk-tera').value) || 1,
    shiny: parseInt(document.getElementById('pk-shiny').value) || 0,
    evs: evs,
    moves: moves
  };
  const res = await fetch('/admin/api/pokemon', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(body)
  });
  const data = await res.json();
  renderState(data.state);
  showToast('✓ Pokémon salvo na Box com IVs 6x31!');
}

async function deletePokemon(uid) {
  const res = await fetch('/admin/api/pokemon/delete', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({ uid })
  });
  const data = await res.json();
  renderState(data.state);
  showToast('🗑️ Pokémon ' + uid + ' removido da Box.');
}

function editTeam(tid) {
  const t = (currentState.teams || {})[String(tid)];
  if (!t) return;
  document.getElementById('tm-tid').value = t.tid;
  document.getElementById('tm-nam').value = t.nam || '';
  document.getElementById('tm-active').value = String(t.cuflg || 0);
  document.getElementById('tm-mem').value = (t.mem || []).join(',');
}

async function saveTeam() {
  const tid = parseInt(document.getElementById('tm-tid').value) || 1;
  const active = parseInt(document.getElementById('tm-active').value) || 0;
  const mem = document.getElementById('tm-mem').value.split(',').map(s => s.trim()).filter(Boolean);
  const res = await fetch('/admin/api/team', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
      tid: tid,
      nam: document.getElementById('tm-nam').value.trim() || ('Team ' + tid),
      cuflg: active,
      dcuflg: active,
      mem: mem
    })
  });
  const data = await res.json();
  renderState(data.state);
  showToast('✓ Time #' + tid + ' atualizado!');
}

async function deleteTeam(tid) {
  const res = await fetch('/admin/api/team/delete', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({ tid })
  });
  const data = await res.json();
  renderState(data.state);
  showToast('🗑️ Time #' + tid + ' removido.');
}

function editBot(id) {
  const b = (currentState.bots || []).find(x => x.id === id);
  if (!b) return;
  document.getElementById('bot-id').value = b.id;
  document.getElementById('bot-name').value = b.trainer_name;
  document.getElementById('bot-ra').value = b.ra;
  document.getElementById('bot-mons').value = (b.mons || []).join(',');
}

async function saveBot() {
  const mons = document.getElementById('bot-mons').value.split(',').map(x => parseInt(x.trim()) || 0).filter(x => x > 0);
  const res = await fetch('/admin/api/bot', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
      id: parseInt(document.getElementById('bot-id').value) || 0,
      trainer_name: document.getElementById('bot-name').value.trim() || 'Champion Bot',
      ra: parseInt(document.getElementById('bot-ra').value) || 1800,
      mons: mons
    })
  });
  const data = await res.json();
  renderState(data.state);
  showToast('🤖 Bot de Matchmaking salvo!');
}

async function deleteBot(id) {
  const res = await fetch('/admin/api/bot/delete', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({ id })
  });
  const data = await res.json();
  renderState(data.state);
  showToast('🗑️ Bot removido.');
}

async function resetAccount() {
  if (!confirm('Deseja restaurar o perfil e a Box para o estado inicial padrão?')) return;
  const res = await fetch('/admin/api/reset', { method: 'POST' });
  const data = await res.json();
  renderState(data.state);
  showToast('🔄 Conta restaurada para o padrão inicial!');
}

async function loadTraffic() {
  const res = await fetch('/admin/api/traffic');
  const data = await res.json();
  if (data.public_url) {
    document.getElementById('server-url-label').innerText = data.public_url;
  }
  const logs = data.logs || [];
  const tbody = document.getElementById('traffic-body');
  if (!logs.length) {
    tbody.innerHTML = '<tr><td colspan="5" style="color:var(--muted)">Nenhuma requisição de jogo registrada ainda nesta sessão.</td></tr>';
    return;
  }
  tbody.innerHTML = logs.map(l =>
    '<tr>' +
      '<td><code>' + l.time + '</code></td>' +
      '<td><span class="method-badge">' + l.method + '</span></td>' +
      '<td><strong>' + l.path + '</strong></td>' +
      '<td>' + l.duration_ms + ' ms</td>' +
      '<td style="color:var(--muted);font-size:12px">' + (l.note || '200 OK') + '</td>' +
    '</tr>'
  ).join('');
}

loadAll();
setInterval(loadTraffic, 5000);
</script>
</body>
</html>`
