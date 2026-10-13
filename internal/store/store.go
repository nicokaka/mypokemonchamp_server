package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nicokaka/mypokechamp_server/internal/model"
)

type PlayerState struct {
	UUID      string                                  `json:"uuid"`
	SupportID string                                  `json:"support_id"`
	DeviceCd  string                                  `json:"device_code"`
	CsrfToken string                                  `json:"csrf_token"`
	SessionID string                                  `json:"session_id"`
	UdVer     int                                     `json:"ud_ver"`
	Profile   model.ApiUserProfileParam               `json:"profile"`
	Main      model.ApiUserMainParam                  `json:"main"`
	Avatar    model.ApiUserAvatarParam                `json:"avatar"`
	Pokemons  map[string]model.ApiUserPokemonParam    `json:"pokemons"`
	Teams     map[string]model.ApiUserTeamParam       `json:"teams"`
	Trainings map[string]model.ApiUserTrainingCodeParam `json:"trainings"`
	Bots      []model.MdBotMergeParam                 `json:"bots"`
}

type Store struct {
	mu       sync.RWMutex
	filePath string
	state    PlayerState
}

func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	fp := filepath.Join(dataDir, "player_state.json")
	s := &Store{filePath: fp}
	if data, err := os.ReadFile(fp); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &s.state); err == nil && len(s.state.Pokemons) > 0 {
			return s, nil
		}
	}
	s.state = defaultPlayerState()
	_ = s.Save()
	return s, nil
}

func (s *Store) Save() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.state, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *Store) GetState() PlayerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *Store) UpdateAuth(uuid, name, csrfToken, sessionID string) PlayerState {
	s.mu.Lock()
	if uuid != "" {
		s.state.UUID = uuid
		s.state.Profile.Id = uuid
	}
	if name != "" {
		s.state.Profile.Nam = name
	}
	if csrfToken != "" {
		s.state.CsrfToken = csrfToken
	}
	if sessionID != "" {
		s.state.SessionID = sessionID
	}
	s.state.Profile.Llog = uint64(time.Now().Unix())
	st := s.state
	s.mu.Unlock()
	_ = s.Save()
	return st
}

func (s *Store) BuildFullUD() *model.ApiUdParam {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prf := s.state.Profile
	main := s.state.Main
	main.Pcnt = len(s.state.Pokemons)
	avat := s.state.Avatar

	pokes := make(map[string]model.ApiUserPokemonParam, len(s.state.Pokemons))
	for k, v := range s.state.Pokemons {
		pokes[k] = v
	}
	teams := make(map[string]model.ApiUserTeamParam, len(s.state.Teams))
	for k, v := range s.state.Teams {
		teams[k] = v
	}
	tngs := make(map[string]model.ApiUserTrainingCodeParam, len(s.state.Trainings))
	for k, v := range s.state.Trainings {
		tngs[k] = v
	}

	return &model.ApiUdParam{
		Prf:     &model.UdUpdate[model.ApiUserProfileParam]{All: &prf},
		Main:    &model.UdUpdate[model.ApiUserMainParam]{All: &main},
		Avat:    &model.UdUpdate[model.ApiUserAvatarParam]{All: &avat},
		Pokes:   &model.UdDiffMap[model.ApiUserPokemonParam]{All: pokes},
		Team:    &model.UdDiffMap[model.ApiUserTeamParam]{All: teams},
		Tng:     &model.UdDiffMap[model.ApiUserTrainingCodeParam]{All: tngs},
		Label:   map[string]interface{}{"all": map[string]interface{}{}},
		Ssndt:   map[string]interface{}{"all": map[string]interface{}{}},
		Btrk:    map[string]interface{}{"all": map[string]interface{}{}},
		Club:    map[string]interface{}{"all": map[string]interface{}{}},
		Avatp:   map[string]interface{}{"all": map[string]interface{}{}},
		Invent:  map[string]interface{}{"all": defaultInventory()},
		Miss:    map[string]interface{}{"all": map[string]interface{}{}},
		Tachv:   map[string]interface{}{"all": map[string]interface{}{}},
		Pachv:   map[string]interface{}{"all": map[string]interface{}{}},
		Shopdt:  map[string]interface{}{"all": map[string]interface{}{}},
		Prfsumdt: map[string]interface{}{"all": map[string]interface{}{"dtlst": []interface{}{}}},
		Usedhis: map[string]interface{}{"all": map[string]interface{}{}},
		Freebox: map[string]interface{}{"all": map[string]interface{}{}},
		UdVer:   s.state.UdVer,
	}
}

func (s *Store) AddOrUpdatePokemon(uid string, p model.ApiUserPokemonParam) *model.ApiUdParam {
	s.mu.Lock()
	if uid == "" {
		uid = fmt.Sprintf("pk_custom_%d", len(s.state.Pokemons)+1)
	}
	p.A1 = uid
	if p.A0 == "" {
		p.A0 = s.state.Profile.TrnId
	}
	now := uint64(time.Now().Unix())
	if p.A6 == 0 {
		p.A6 = now
	}
	p.A7 = now
	if len(p.Bf) < 6 {
		p.Bf = []byte{31, 31, 31, 31, 31, 31}
	}
	if len(p.Bg) < 4 {
		p.Bg = []int{3, 3, 3, 3}
	}
	if len(p.Bh) == 0 && len(p.B6) > 0 {
		p.Bh = append([]uint16(nil), p.B6...)
	}
	if p.Bi == 0 {
		p.Bi = p.B5
	}
	if p.Bx == "" {
		p.Bx = s.state.Profile.Nam
	}
	if p.I == 0 {
		p.I = len(s.state.Pokemons) + 1
	}
	s.state.Pokemons[uid] = p
	s.state.Main.Pcnt = len(s.state.Pokemons)
	s.state.UdVer++
	s.mu.Unlock()
	_ = s.Save()
	return s.BuildFullUD()
}

func (s *Store) ApplyTraining(uid string, hp, atk, def, agi, spatk, spdef byte, toku uint16, sei byte, waza []uint16) *model.ApiUdParam {
	s.mu.Lock()
	if p, ok := s.state.Pokemons[uid]; ok {
		p.B9 = hp
		p.Ba = atk
		p.Bb = def
		p.Bc = agi
		p.Bd = spatk
		p.Be = spdef
		if toku > 0 {
			p.B5 = toku
		}
		p.Bj = sei
		p.Bk = sei
		if len(waza) > 0 {
			p.B6 = waza
		}
		p.A7 = uint64(time.Now().Unix())
		s.state.Pokemons[uid] = p
		s.state.UdVer++
	}
	s.mu.Unlock()
	_ = s.Save()
	return s.BuildFullUD()
}

func (s *Store) UpdateTeam(tid string, t model.ApiUserTeamParam) *model.ApiUdParam {
	s.mu.Lock()
	if tid == "" {
		tid = "team_01"
	}
	t.Tid = tid
	s.state.Teams[tid] = t
	s.state.UdVer++
	s.mu.Unlock()
	_ = s.Save()
	return s.BuildFullUD()
}

func defaultInventory() map[string]interface{} {
	now := uint64(time.Now().Unix())
	items := map[string]interface{}{}
	// Pre-unlock Mega Stones, Z-Crystals, Tera Shards, and competitive held items (IDs 1..300)
	competitiveItems := []int{
		1, 2, 3, 4, 15, 197, 213, 214, 220, 230, 234, 236, 242, 245, 247, 253, 268, 270, 275,
		640, 644, 656, 657, 658, 659, 660, 661, 662, 663, 666, 670, 671, 675, 676, 679, 752, 759,
	}
	for _, id := range competitiveItems {
		items[fmt.Sprintf("%d", id)] = map[string]interface{}{
			"c":   99,
			"upd": now,
		}
	}
	return items
}

func makeCompetitiveMon(uid string, slot int, monsNo uint16, nickname string, ability uint16, item uint16, nature byte, teraType int, shiny byte, evs [6]byte, moves []uint16) model.ApiUserPokemonParam {
	now := uint64(time.Now().Unix())
	return model.ApiUserPokemonParam{
		Ver: 1,
		A0:  "CHAMP-2026-0001",
		A1:  uid,
		A5:  0,
		A6:  now,
		A7:  now,
		A8:  0,
		A9:  fmt.Sprintf("HOME-%s", uid),
		Aa:  now,
		B0:  100000 + slot,
		B1:  monsNo,
		B2:  0,
		B3:  0,
		B4:  125000, // Level 50
		B5:  ability,
		B6:  moves,
		B7:  item,
		B8:  4, // Poke Ball
		B9:  evs[0],
		Ba:  evs[1],
		Bb:  evs[2],
		Bc:  evs[3],
		Bd:  evs[4],
		Be:  evs[5],
		Bf:  []byte{31, 31, 31, 31, 31, 31},
		Bg:  []int{3, 3, 3, 3},
		Bh:  moves,
		Bi:  ability,
		Bj:  nature,
		Bk:  nature,
		Bl:  shiny,
		Bm:  teraType,
		Bn:  2, // English/Global
		Bo:  nickname,
		Bp:  0,
		Bt:  128,
		Bu:  128,
		Bv:  []int{1},
		Bw:  0,
		Bx:  "Nicolas",
		By:  0,
		I:   slot,
	}
}

func defaultPlayerState() PlayerState {
	now := uint64(time.Now().Unix())
	pokes := map[string]model.ApiUserPokemonParam{
		"pk_001": makeCompetitiveMon("pk_001", 1, 6, "Charizard", 66, 660, 15, 2, 1, [6]byte{4, 0, 0, 252, 252, 0}, []uint16{53, 403, 76, 182}),       // Charizard (Shiny, Charizardite Y)
		"pk_002": makeCompetitiveMon("pk_002", 2, 445, "Garchomp", 134, 270, 13, 5, 0, [6]byte{4, 252, 0, 252, 0, 0}, []uint16{89, 337, 442, 182}),      // Garchomp (Rough Skin, Life Orb)
		"pk_003": makeCompetitiveMon("pk_003", 3, 448, "Lucario", 39, 675, 13, 2, 1, [6]byte{4, 252, 0, 252, 0, 0}, []uint16{370, 418, 245, 14}),        // Lucario (Shiny, Lucarionite)
		"pk_004": makeCompetitiveMon("pk_004", 4, 94, "Gengar", 130, 275, 15, 8, 1, [6]byte{4, 0, 0, 252, 252, 0}, []uint16{247, 188, 524, 182}),        // Gengar (Shiny, Focus Sash)
		"pk_005": makeCompetitiveMon("pk_005", 5, 149, "Dragonite", 136, 234, 3, 1, 0, [6]byte{252, 252, 0, 4, 0, 0}, []uint16{245, 89, 7, 349}),        // Dragonite (Multiscale, Leftovers)
		"pk_006": makeCompetitiveMon("pk_006", 6, 376, "Metagross", 29, 759, 3, 9, 1, [6]byte{252, 252, 0, 4, 0, 0}, []uint16{309, 418, 428, 8}),        // Metagross (Shiny, Metagrossite)
		"pk_007": makeCompetitiveMon("pk_007", 7, 25, "Pikachu", 31, 236, 15, 4, 0, [6]byte{4, 0, 0, 252, 252, 0}, []uint16{85, 521, 98, 182}),          // Pikachu (Light Ball)
		"pk_008": makeCompetitiveMon("pk_008", 8, 282, "Gardevoir", 36, 657, 15, 18, 1, [6]byte{4, 0, 0, 252, 252, 0}, []uint16{585, 94, 473, 182}),     // Gardevoir (Shiny, Gardevoirite)
		"pk_009": makeCompetitiveMon("pk_009", 9, 658, "Greninja", 168, 270, 15, 3, 1, [6]byte{4, 0, 0, 252, 252, 0}, []uint16{56, 399, 58, 421}),       // Greninja (Shiny, Protean)
		"pk_010": makeCompetitiveMon("pk_010", 10, 248, "Tyranitar", 45, 640, 3, 6, 0, [6]byte{252, 252, 0, 0, 0, 4}, []uint16{444, 242, 89, 182}),     // Tyranitar (Sand Stream, Assault Vest)
		"pk_011": makeCompetitiveMon("pk_011", 11, 373, "Salamence", 22, 769, 13, 10, 0, [6]byte{4, 252, 0, 252, 0, 0}, []uint16{38, 89, 355, 349}),    // Salamence (Intimidate, Salamencite)
		"pk_012": makeCompetitiveMon("pk_012", 12, 681, "Aegislash", 176, 234, 17, 9, 1, [6]byte{252, 0, 0, 0, 252, 4}, []uint16{247, 430, 425, 588}),  // Aegislash (Shiny, Stance Change)
	}

	teams := map[string]model.ApiUserTeamParam{
		"team_01": {
			Tid:    "team_01",
			Nam:    "Champions Main",
			Lng:    2,
			Mem:    []string{"pk_001", "pk_002", "pk_003", "pk_004", "pk_005", "pk_006"},
			Cuflg:  1,
			Dcuflg: 1,
			Itms: []model.ApiUserTeamItemParam{
				{P: "pk_001", I: 660},
				{P: "pk_002", I: 270},
				{P: "pk_003", I: 675},
				{P: "pk_004", I: 275},
				{P: "pk_005", I: 234},
				{P: "pk_006", I: 759},
			},
		},
		"team_02": {
			Tid:    "team_02",
			Nam:    "Kalos & Hoenn",
			Lng:    2,
			Mem:    []string{"pk_007", "pk_008", "pk_009", "pk_010", "pk_011", "pk_012"},
			Cuflg:  0,
			Dcuflg: 0,
			Itms: []model.ApiUserTeamItemParam{
				{P: "pk_007", I: 236},
				{P: "pk_008", I: 657},
				{P: "pk_009", I: 270},
				{P: "pk_010", I: 640},
				{P: "pk_011", I: 769},
				{P: "pk_012", I: 234},
			},
		},
	}

	bots := []model.MdBotMergeParam{
		{ID: 1001, TrainerName: "Champion Cynthia", Use: 1, Lose: 0, Ra: 1850, Ira: 1850, Avatar: 101, Mons: []uint16{445, 448, 350, 468, 442, 407}},
		{ID: 1002, TrainerName: "Champion Leon", Use: 1, Lose: 0, Ra: 1800, Ira: 1800, Avatar: 102, Mons: []uint16{6, 681, 887, 464, 612, 812}},
		{ID: 1003, TrainerName: "Champion Red", Use: 1, Lose: 0, Ra: 1900, Ira: 1900, Avatar: 103, Mons: []uint16{25, 6, 3, 9, 143, 131}},
		{ID: 1004, TrainerName: "Champion Steven", Use: 1, Lose: 0, Ra: 1750, Ira: 1750, Avatar: 104, Mons: []uint16{376, 227, 306, 344, 346, 348}},
		{ID: 1005, TrainerName: "Rival Blue", Use: 1, Lose: 0, Ra: 1700, Ira: 1700, Avatar: 105, Mons: []uint16{18, 65, 112, 103, 130, 59}},
	}

	return PlayerState{
		UUID:      "champ-uuid-nicolas-0001",
		SupportID: "SUP-CHAMP-9999",
		DeviceCd:  "DVC-CHAMP-0001",
		CsrfToken: "tok_champ_private_0001",
		SessionID: "sid_champ_private_0001",
		UdVer:     1,
		Profile: model.ApiUserProfileParam{
			AgTA:  now,
			Bth:   200001,
			Clogc: 1,
			Cou:   31, // Brazil/Global
			CreAt: now,
			Flng:  2,
			Fri:   "1234-5678-9012",
			Id:    "champ-uuid-nicolas-0001",
			Lavsp: 1,
			Lavsw: 1,
			Llday: now,
			Lldaz: now,
			Llog:  now,
			Lng:   2,
			Mode:  1,
			Motim: now,
			Nam:   "Nicolas",
			PpV:   1,
			Push:  1,
			Sup:   "SUP-CHAMP-9999",
			Tlogc: 10,
			TmV:   1,
			Tzof:  -180,
			Dv:    "Android",
			Os:    "Android 14",
			Trtit: 1,
			Trico: 1,
			Trbad: 1,
			TrnId: "CHAMP-2026-0001",
			TrCAt: now,
			TrIAt: now,
		},
		Main: model.ApiUserMainParam{
			Vp:     999999, // Unlimited VP for free training & customization
			TuPrg:  999,    // Tutorial completed
			TuFlgs: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			Pbsiz:  300,    // 300 Box capacity
			Pbexc:  10,
			Pcnt:   len(pokes),
			Ssnno:  1,
			Bpasi:  map[string]interface{}{"typ": 2, "stt": 1, "exp": 50000},
			Subsc:  map[string]interface{}{"typ": 1, "fpt": now, "lst": now, "cst": 1, "exp": now + 31536000, "pmdid": "sub_premium"},
			Misri:  map[string]interface{}{"rd": now + 86400, "mid": []int{}, "rrc": 3},
			Emisi:  map[string]interface{}{"end": now + 604800, "rrc": 3},
			Putoi:  []interface{}{},
			Ptids:  []int{},
			Mxrnk:  10,
			Fcbs:   300,
			Fpbs:   300,
			Swpai:  map[string]interface{}{"pmids": []string{}, "addbs": 300},
		},
		Avatar: model.ApiUserAvatarParam{
			Fac: map[string]int{"p0": 1, "p0c": 1, "p1": 1, "p2": 1, "p2c": 1, "p3": 1, "p4": 1, "p5": 1, "p6": 1, "p7": 1, "p7c": 1, "p8c": 1, "p8l": 1},
			Cos: map[string]int{"p0": 1, "p0c": 1, "p1": 1, "p2": 1, "p3": 1, "p4": 1, "p5": 1, "p6": 1, "p7": 1, "p8": 1, "p9": 1, "pa": 1, "pb": 1},
			Mot: map[string]int{"p0": 1, "p1": 1, "p2": 1},
		},
		Pokemons:  pokes,
		Teams:     teams,
		Trainings: map[string]model.ApiUserTrainingCodeParam{},
		Bots:      bots,
	}
}
