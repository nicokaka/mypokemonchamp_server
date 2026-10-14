package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/nicokaka/mypokechamp_server/internal/model"
)

type PlayerState struct {
	UUID         string                                    `json:"uuid"`
	SupportID    string                                    `json:"support_id"`
	DeviceCd     string                                    `json:"device_code"`
	CsrfToken    string                                    `json:"csrf_token"`
	SessionID    string                                    `json:"session_id"`
	UdVer        int64                                     `json:"ud_ver"`
	SchemaVer    int                                       `json:"schema_ver"`
	RankSingle   int                                       `json:"rank_single"`
	RatingSingle float64                                   `json:"rating_single"`
	WinsSingle   int                                       `json:"wins_single"`
	LossesSingle int                                       `json:"losses_single"`
	RankDouble   int                                       `json:"rank_double"`
	RatingDouble float64                                   `json:"rating_double"`
	WinsDouble   int                                       `json:"wins_double"`
	LossesDouble int                                       `json:"losses_double"`
	SeasonPoints int                                       `json:"season_points"`
	Profile      model.ApiUserProfileParam                 `json:"profile"`
	Main         model.ApiUserMainParam                    `json:"main"`
	Avatar       model.ApiUserAvatarParam                  `json:"avatar"`
	Pokemons     map[string]model.ApiUserPokemonParam      `json:"pokemons"`
	Teams        map[string]model.ApiUserTeamParam         `json:"teams"`
	Trainings    map[string]model.ApiUserTrainingCodeParam `json:"trainings"`
	Bots         []model.MdBotMergeParam                   `json:"bots"`
}

const (
	currentSchemaVer = 2
	defaultUUID42    = "champ0001nicolas0001pokemonchampions000001"
)

func normalizeUUID42(u string) string {
	if len(u) == 42 {
		return u
	}
	if u == "" {
		return defaultUUID42
	}
	if len(u) > 42 {
		return u[:42]
	}
	pad := defaultUUID42
	return (u + pad)[:42]
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
		if err := json.Unmarshal(data, &s.state); err == nil &&
			s.state.SchemaVer == currentSchemaVer &&
			len(s.state.Pokemons) > 0 {
			s.state.UUID = normalizeUUID42(s.state.UUID)
			s.state.Profile.Id = s.state.UUID
			s.state.UdVer = 1
			if s.state.RankSingle == 0 {
				s.state.RankSingle = 10
				s.state.RatingSingle = 1500
				s.state.WinsSingle = 10
			}
			if s.state.RankDouble == 0 {
				s.state.RankDouble = 10
				s.state.RatingDouble = 1500
				s.state.WinsDouble = 10
			}
			if s.state.SeasonPoints == 0 {
				s.state.SeasonPoints = 1000
			}
			_ = s.Save()
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
	if len(uuid) == 42 {
		s.state.UUID = uuid
		s.state.Profile.Id = uuid
	} else if s.state.UUID == "" || len(s.state.UUID) != 42 {
		s.state.UUID = normalizeUUID42(uuid)
		s.state.Profile.Id = s.state.UUID
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
	s.state.UdVer = 1
	s.state.Profile.Llog = time.Now().Unix()
	st := s.state
	s.mu.Unlock()
	_ = s.Save()
	return st
}

func (s *Store) BuildFullUD() *model.ApiUdParam {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prf := s.state.Profile
	if prf.Trbad == nil {
		prf.Trbad = []string{}
	}
	main := s.state.Main
	main.Pcnt = len(s.state.Pokemons)
	if main.TuFlgs == nil {
		main.TuFlgs = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	}
	if main.Putoi == nil {
		main.Putoi = map[string]map[string]int{}
	}
	if main.Ptids == nil {
		main.Ptids = []interface{}{}
	}
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
		Prf:      &model.UdUpdate[model.ApiUserProfileParam]{All: &prf},
		Main:     &model.UdUpdate[model.ApiUserMainParam]{All: &main},
		Avat:     &model.UdUpdate[model.ApiUserAvatarParam]{All: &avat},
		Pokes:    &model.UdDiffMap[model.ApiUserPokemonParam]{All: pokes},
		Team:     &model.UdDiffMap[model.ApiUserTeamParam]{All: teams},
		Tng:      &model.UdDiffMap[model.ApiUserTrainingCodeParam]{All: tngs},
		Label:    map[string]interface{}{"all": map[string]interface{}{}},
		Ssndt:    map[string]interface{}{"all": buildSeasonData(s.state)},
		Btrk:     map[string]interface{}{"all": map[string]interface{}{"cdt": []interface{}{}}},
		Club:     map[string]interface{}{"all": map[string]interface{}{}},
		Avatp:    map[string]interface{}{"all": map[string]interface{}{}},
		Invent:   map[string]interface{}{"all": defaultInventory()},
		Miss:     map[string]interface{}{"all": map[string]interface{}{}},
		Tachv:    map[string]interface{}{"all": map[string]interface{}{}},
		Pachv:    map[string]interface{}{"all": map[string]interface{}{}},
		Shopdt:   map[string]interface{}{"all": map[string]interface{}{}},
		Prfsumdt: map[string]interface{}{"all": map[string]interface{}{"dtlst": map[string]int{}}},
		Usedhis:  map[string]interface{}{"all": map[string]interface{}{}},
		Freebox:  map[string]interface{}{"all": []string{}},
		UdVer:    1,
	}
}

func (s *Store) AddOrUpdatePokemon(uid string, p model.ApiUserPokemonParam) *model.ApiUdParam {
	s.mu.Lock()
	if uid == "" {
		uid = fmt.Sprintf("pk_custom_%d", len(s.state.Pokemons)+1)
	}
	p.A1 = uid
	if p.Ver == 0 {
		p.Ver = 1
	}
	if p.A0 == "" {
		p.A0 = s.state.Profile.TrnId
	}
	now := uint64(time.Now().Unix())
	if p.A6 == 0 {
		p.A6 = now
	}
	p.A7 = now
	if len(p.Bf) < 6 {
		p.Bf = []uint16{31, 31, 31, 31, 31, 31}
	}
	if len(p.Bg) < 4 {
		p.Bg = []int64{3, 3, 3, 3}
	}
	if len(p.Bh) == 0 && len(p.B6) > 0 {
		p.Bh = append([]int16(nil), p.B6...)
	}
	if len(p.Bv) == 0 {
		p.Bv = []int64{1}
	}
	if p.Bi == 0 {
		p.Bi = p.B5
	}
	if p.Bx == "" {
		p.Bx = s.state.Profile.Nam
	}
	if p.I == 0 {
		p.I = uint32(len(s.state.Pokemons) + 1)
	}
	s.state.Pokemons[uid] = p
	s.state.Main.Pcnt = len(s.state.Pokemons)
	s.state.UdVer++
	s.mu.Unlock()
	_ = s.Save()
	return s.BuildFullUD()
}

func (s *Store) ApplyTraining(uid string, hp, atk, def, agi, spatk, spdef byte, toku uint16, sei byte, waza []int16) *model.ApiUdParam {
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

func (s *Store) UpdateTeam(tid int, t model.ApiUserTeamParam) *model.ApiUdParam {
	s.mu.Lock()
	if tid <= 0 {
		tid = 1
	}
	t.Tid = tid
	if t.Itms == nil {
		t.Itms = []model.ApiUserTeamItemParam{}
	}
	if t.Mem == nil {
		t.Mem = []string{}
	}
	s.state.Teams[strconv.Itoa(tid)] = t
	s.state.UdVer++
	s.mu.Unlock()
	_ = s.Save()
	return s.BuildFullUD()
}

func (s *Store) UpdateTeams(teams []model.ApiUserTeamParam) *model.ApiUdParam {
	s.mu.Lock()
	for i, t := range teams {
		if t.Tid <= 0 {
			t.Tid = i + 1
		}
		if t.Itms == nil {
			t.Itms = []model.ApiUserTeamItemParam{}
		}
		if t.Mem == nil {
			t.Mem = []string{}
		}
		s.state.Teams[strconv.Itoa(t.Tid)] = t
	}
	s.state.UdVer++
	s.mu.Unlock()
	_ = s.Save()
	return s.BuildFullUD()
}

func (s *Store) DeleteTeam(tid int) *model.ApiUdParam {
	s.mu.Lock()
	delete(s.state.Teams, strconv.Itoa(tid))
	s.state.UdVer++
	s.mu.Unlock()
	_ = s.Save()
	return s.BuildFullUD()
}

func (s *Store) UpdateProfile(nam, trtit, trico string, trbad []string) *model.ApiUdParam {
	s.mu.Lock()
	if nam != "" {
		s.state.Profile.Nam = nam
	}
	if trtit != "" {
		s.state.Profile.Trtit = trtit
	}
	if trico != "" {
		s.state.Profile.Trico = trico
	}
	if trbad != nil {
		s.state.Profile.Trbad = trbad
	}
	s.state.UdVer++
	s.mu.Unlock()
	_ = s.Save()
	return s.BuildFullUD()
}

func (s *Store) UpdateAvatar(avat model.ApiUserAvatarParam) *model.ApiUdParam {
	s.mu.Lock()
	if len(avat.Fac) > 0 {
		s.state.Avatar.Fac = avat.Fac
	}
	if len(avat.Cos) > 0 {
		s.state.Avatar.Cos = avat.Cos
	}
	if len(avat.Mot) > 0 {
		s.state.Avatar.Mot = avat.Mot
	}
	s.state.UdVer++
	s.mu.Unlock()
	_ = s.Save()
	return s.BuildFullUD()
}

func (s *Store) UpdateNickname(uid, nickname string) *model.ApiUdParam {
	s.mu.Lock()
	if p, ok := s.state.Pokemons[uid]; ok {
		p.Bo = nickname
		if nickname != "" {
			p.Bp = 1
		}
		p.A7 = uint64(time.Now().Unix())
		s.state.Pokemons[uid] = p
		s.state.UdVer++
	}
	s.mu.Unlock()
	_ = s.Save()
	return s.BuildFullUD()
}

func (s *Store) UpdateRibbon(uid string, ribbon byte) *model.ApiUdParam {
	s.mu.Lock()
	if p, ok := s.state.Pokemons[uid]; ok {
		p.Bw = ribbon
		p.A7 = uint64(time.Now().Unix())
		s.state.Pokemons[uid] = p
		s.state.UdVer++
	}
	s.mu.Unlock()
	_ = s.Save()
	return s.BuildFullUD()
}

func (s *Store) DeletePokemons(uids []string) *model.ApiUdParam {
	s.mu.Lock()
	for _, uid := range uids {
		delete(s.state.Pokemons, uid)
	}
	s.state.Main.Pcnt = len(s.state.Pokemons)
	s.state.UdVer++
	s.mu.Unlock()
	_ = s.Save()
	return s.BuildFullUD()
}

func (s *Store) BuildPlayerBattleTeam() model.ApiTeamParam {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var selectedTeam model.ApiUserTeamParam
	found := false
	for _, t := range s.state.Teams {
		if t.Cuflg == 1 || !found {
			selectedTeam = t
			found = true
			if t.Cuflg == 1 {
				break
			}
		}
	}

	var tme []model.ApiUserPokemonParam
	for _, uid := range selectedTeam.Mem {
		if p, ok := s.state.Pokemons[uid]; ok {
			tme = append(tme, p)
		}
	}
	if len(tme) == 0 {
		for _, p := range s.state.Pokemons {
			tme = append(tme, p)
			if len(tme) == 6 {
				break
			}
		}
	}

	ra := float32(s.state.RatingSingle)
	if ra <= 0 {
		ra = 1500.0
	}
	rk := s.state.RankSingle
	if rk <= 0 {
		rk = 10
	}
	wic := s.state.WinsSingle
	if wic <= 0 {
		wic = 10
	}
	trbad := s.state.Profile.Trbad
	if trbad == nil {
		trbad = []string{"1"}
	}
	return model.ApiTeamParam{
		Fri:   s.state.Profile.Fri,
		Nam:   s.state.Profile.Nam,
		Lng:   s.state.Profile.Lng,
		Cou:   s.state.Profile.Cou,
		Trtit: s.state.Profile.Trtit,
		Trico: s.state.Profile.Trico,
		Trbad: trbad,
		Rk:    int16(rk),
		Wic:   wic,
		Rat:   model.ApiRateParam{Ra: &ra},
		Par:   0,
		Tme:   tme,
		Bamin: 900,
		Ava:   s.state.Avatar,
		Tse:   []int{0, 1, 2, 3},
	}
}

func (s *Store) BuildBotBattleTeam(botIdx int) model.ApiTeamParam {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bot := model.MdBotMergeParam{
		ID:          1001,
		TrainerName: "Champion Cynthia",
		Ra:          1850,
		Mons:        []uint16{445, 448, 350, 468, 442, 407},
	}
	if len(s.state.Bots) > 0 {
		if botIdx < 0 || botIdx >= len(s.state.Bots) {
			botIdx = 0
		}
		bot = s.state.Bots[botIdx]
	}

	tme := make([]model.ApiUserPokemonParam, 0, len(bot.Mons))
	for i, monsNo := range bot.Mons {
		uid := fmt.Sprintf("bot_%d_pk_%d", bot.ID, i+1)
		mon := makeCompetitiveMon(
			uid,
			i+1,
			monsNo,
			fmt.Sprintf("Mon-%d", monsNo),
			134,
			270,
			13,
			1,
			0,
			[6]byte{4, 252, 0, 252, 0, 0},
			[]int16{89, 337, 442, 182},
		)
		mon.Bx = bot.TrainerName
		tme = append(tme, mon)
	}

	ra := float32(bot.Ra)
	return model.ApiTeamParam{
		Fri:   fmt.Sprintf("BOT-%d", bot.ID),
		Nam:   bot.TrainerName,
		Lng:   2,
		Cou:   31,
		Trtit: "1",
		Trico: "1",
		Trbad: []string{"1"},
		Rk:    10,
		Wic:   50,
		Rat:   model.ApiRateParam{Ra: &ra},
		Par:   1,
		Tme:   tme,
		Bamin: 900,
		Ava:   s.state.Avatar,
		Tse:   []int{0, 1, 2, 3},
	}
}

type AdminProfileUpdate struct {
	Nam          string   `json:"nam"`
	TrnId        string   `json:"trn_id"`
	Fri          string   `json:"fri"`
	Trtit        string   `json:"trtit"`
	Trico        string   `json:"trico"`
	Trbad        []string `json:"trbad"`
	Cou          int      `json:"cou"`
	Lng          int      `json:"lng"`
	Vp           int      `json:"vp"`
	Pbsiz        int      `json:"pbsiz"`
	SeasonPoints int      `json:"season_points"`
	RankSingle   int      `json:"rank_single"`
	RatingSingle float64  `json:"rating_single"`
	WinsSingle   int      `json:"wins_single"`
	LossesSingle int      `json:"losses_single"`
	RankDouble   int      `json:"rank_double"`
	RatingDouble float64  `json:"rating_double"`
	WinsDouble   int      `json:"wins_double"`
	LossesDouble int      `json:"losses_double"`
}

func (s *Store) UpdateProfileAdmin(u AdminProfileUpdate) PlayerState {
	s.mu.Lock()
	if u.Nam != "" {
		s.state.Profile.Nam = u.Nam
	}
	if u.TrnId != "" {
		s.state.Profile.TrnId = u.TrnId
	}
	if u.Fri != "" {
		s.state.Profile.Fri = u.Fri
	}
	if u.Trtit != "" {
		s.state.Profile.Trtit = u.Trtit
	}
	if u.Trico != "" {
		s.state.Profile.Trico = u.Trico
	}
	if len(u.Trbad) > 0 {
		s.state.Profile.Trbad = u.Trbad
	}
	if u.Cou > 0 {
		s.state.Profile.Cou = u.Cou
	}
	if u.Lng > 0 {
		s.state.Profile.Lng = u.Lng
		s.state.Profile.Flng = u.Lng
	}
	if u.Vp >= 0 {
		s.state.Main.Vp = int64(u.Vp)
	}
	if u.Pbsiz > 0 {
		s.state.Main.Pbsiz = u.Pbsiz
		s.state.Main.Fcbs = int64(u.Pbsiz)
		s.state.Main.Fpbs = int64(u.Pbsiz)
	}
	if u.SeasonPoints >= 0 {
		s.state.SeasonPoints = u.SeasonPoints
	}
	if u.RankSingle > 0 {
		s.state.RankSingle = u.RankSingle
		if uint32(u.RankSingle) > s.state.Main.Mxrnk {
			s.state.Main.Mxrnk = uint32(u.RankSingle)
		}
	}
	if u.RatingSingle > 0 {
		s.state.RatingSingle = u.RatingSingle
	}
	if u.WinsSingle >= 0 {
		s.state.WinsSingle = u.WinsSingle
	}
	if u.LossesSingle >= 0 {
		s.state.LossesSingle = u.LossesSingle
	}
	if u.RankDouble > 0 {
		s.state.RankDouble = u.RankDouble
	}
	if u.RatingDouble > 0 {
		s.state.RatingDouble = u.RatingDouble
	}
	if u.WinsDouble >= 0 {
		s.state.WinsDouble = u.WinsDouble
	}
	if u.LossesDouble >= 0 {
		s.state.LossesDouble = u.LossesDouble
	}
	s.state.UdVer++
	st := s.state
	s.mu.Unlock()
	_ = s.Save()
	return st
}

func (s *Store) MaxOutAccount() PlayerState {
	s.mu.Lock()
	s.state.Main.Vp = 9999999
	s.state.Main.Pbsiz = 900
	s.state.Main.Fcbs = 900
	s.state.Main.Fpbs = 900
	s.state.Main.Mxrnk = 10
	s.state.SeasonPoints = 99999
	s.state.RankSingle = 10
	s.state.RatingSingle = 2100.0
	if s.state.WinsSingle < 100 {
		s.state.WinsSingle = 100
	}
	s.state.RankDouble = 10
	s.state.RatingDouble = 2100.0
	if s.state.WinsDouble < 100 {
		s.state.WinsDouble = 100
	}
	s.state.UdVer++
	st := s.state
	s.mu.Unlock()
	_ = s.Save()
	return st
}

func (s *Store) UnlockMetaPack() int {
	metaMons := []struct {
		uid      string
		monsNo   uint16
		nickname string
		ability  uint16
		item     uint16
		nature   byte
		tera     uint32
		shiny    byte
		evs      [6]byte
		moves    []int16
	}{
		{"pk_meta_150", 150, "Mewtwo", 46, 270, 15, 14, 1, [6]byte{4, 0, 0, 252, 252, 0}, []int16{540, 94, 58, 182}},
		{"pk_meta_384", 384, "Rayquaza", 76, 275, 13, 2, 1, [6]byte{4, 252, 0, 252, 0, 0}, []int16{620, 245, 349, 182}},
		{"pk_meta_382", 382, "Kyogre", 2, 220, 15, 11, 1, [6]byte{4, 0, 0, 252, 252, 0}, []int16{323, 618, 58, 87}},
		{"pk_meta_383", 383, "Groudon", 70, 234, 3, 10, 1, [6]byte{252, 252, 0, 4, 0, 0}, []int16{619, 7, 444, 182}},
		{"pk_meta_888", 888, "Zacian", 234, 1103, 13, 18, 1, [6]byte{4, 252, 0, 252, 0, 0}, []int16{781, 583, 370, 182}},
		{"pk_meta_898", 898, "Calyrex", 267, 275, 15, 8, 0, [6]byte{4, 0, 0, 252, 252, 0}, []int16{824, 94, 417, 182}},
		{"pk_meta_1008", 1008, "Miraidon", 289, 297, 15, 4, 0, [6]byte{4, 0, 0, 252, 252, 0}, []int16{879, 434, 521, 605}},
		{"pk_meta_1007", 1007, "Koraidon", 288, 1699, 13, 10, 0, [6]byte{4, 252, 0, 252, 0, 0}, []int16{878, 394, 888, 182}},
		{"pk_meta_987", 987, "Flutter Mane", 281, 1880, 15, 18, 1, [6]byte{4, 0, 0, 252, 252, 0}, []int16{585, 247, 605, 182}},
		{"pk_meta_892", 892, "Urshifu", 260, 275, 13, 11, 0, [6]byte{4, 252, 0, 252, 0, 0}, []int16{818, 370, 453, 197}},
		{"pk_meta_727", 727, "Incineroar", 22, 1120, 8, 8, 1, [6]byte{252, 4, 156, 0, 0, 96}, []int16{252, 394, 282, 575}},
		{"pk_meta_812", 812, "Rillaboom", 229, 640, 3, 10, 0, [6]byte{252, 252, 0, 4, 0, 0}, []int16{803, 452, 252, 369}},
		{"pk_meta_983", 983, "Kingambit", 128, 240, 3, 2, 1, [6]byte{252, 252, 0, 0, 0, 4}, []int16{869, 389, 442, 182}},
		{"pk_meta_887", 887, "Dragapult", 29, 220, 13, 8, 1, [6]byte{4, 252, 0, 252, 0, 0}, []int16{751, 566, 369, 851}},
	}
	s.mu.Lock()
	added := 0
	for i, m := range metaMons {
		if _, exists := s.state.Pokemons[m.uid]; !exists {
			mon := makeCompetitiveMon(m.uid, len(s.state.Pokemons)+i+1, m.monsNo, m.nickname, m.ability, m.item, m.nature, m.tera, m.shiny, m.evs, m.moves)
			mon.Bx = s.state.Profile.Nam
			s.state.Pokemons[m.uid] = mon
			added++
		}
	}
	s.state.Main.Pcnt = len(s.state.Pokemons)
	s.state.UdVer++
	s.mu.Unlock()
	_ = s.Save()
	return added
}

func (s *Store) SaveBot(bot model.MdBotMergeParam) PlayerState {
	s.mu.Lock()
	if bot.ID <= 0 {
		maxID := 1000
		for _, b := range s.state.Bots {
			if b.ID > maxID {
				maxID = b.ID
			}
		}
		bot.ID = maxID + 1
	}
	if bot.Use == 0 {
		bot.Use = 1
	}
	if bot.Ira == 0 {
		bot.Ira = bot.Ra
	}
	if len(bot.Mons) == 0 {
		bot.Mons = []uint16{6, 445, 448, 94, 149, 376}
	}
	replaced := false
	for i, b := range s.state.Bots {
		if b.ID == bot.ID {
			s.state.Bots[i] = bot
			replaced = true
			break
		}
	}
	if !replaced {
		s.state.Bots = append(s.state.Bots, bot)
	}
	st := s.state
	s.mu.Unlock()
	_ = s.Save()
	return st
}

func (s *Store) DeleteBot(id int) PlayerState {
	s.mu.Lock()
	next := make([]model.MdBotMergeParam, 0, len(s.state.Bots))
	for _, b := range s.state.Bots {
		if b.ID != id {
			next = append(next, b)
		}
	}
	s.state.Bots = next
	st := s.state
	s.mu.Unlock()
	_ = s.Save()
	return st
}

func (s *Store) ResetToDefault() PlayerState {
	s.mu.Lock()
	s.state = defaultPlayerState()
	st := s.state
	s.mu.Unlock()
	_ = s.Save()
	return st
}

func buildSeasonData(st PlayerState) map[string]interface{} {
	rkS := st.RankSingle
	if rkS <= 0 {
		rkS = 10
	}
	ratS := st.RatingSingle
	if ratS <= 0 {
		ratS = 1500.0
	}
	rkD := st.RankDouble
	if rkD <= 0 {
		rkD = 10
	}
	ratD := st.RatingDouble
	if ratD <= 0 {
		ratD = 1500.0
	}
	sp := st.SeasonPoints
	if sp <= 0 {
		sp = 1000
	}
	return map[string]interface{}{
		"1": map[string]interface{}{
			"ssn": 1,
			"tsp": sp,
			"sp":  sp,
			"bpri": map[string]interface{}{
				"st":  1,
				"sta": []uint16{1, 2, 3},
			},
			"rbsi": map[string]interface{}{
				"rk":   rkS,
				"mrk":  rkS,
				"po":   int(ratS),
				"rat":  ratS,
				"win":  st.WinsSingle,
				"los":  st.LossesSingle,
				"dra":  0,
				"podt": map[string]interface{}{},
			},
			"rbdi": map[string]interface{}{
				"rk":   rkD,
				"mrk":  rkD,
				"po":   int(ratD),
				"rat":  ratD,
				"win":  st.WinsDouble,
				"los":  st.LossesDouble,
				"dra":  0,
				"podt": map[string]interface{}{},
			},
			"pp": 1,
		},
	}
}

// defaultInventory returns Net.ApiUserInventoryItemDiffParams.all:
// Dictionary<short, Dictionary<int, ApiUserInventoryItemNullableParam>>
func defaultInventory() map[string]interface{} {
	now := time.Now().Unix()
	heldItems := map[string]interface{}{}
	competitiveItems := []int{
		1, 2, 3, 4, 15, 197, 213, 214, 220, 230, 234, 236, 242, 245, 247, 253, 268, 270, 275,
		640, 644, 656, 657, 658, 659, 660, 661, 662, 663, 666, 670, 671, 675, 676, 679, 752, 759, 769,
	}
	for _, id := range competitiveItems {
		heldItems[strconv.Itoa(id)] = map[string]interface{}{
			"c":   99,
			"upd": now,
		}
	}
	return map[string]interface{}{
		"1": heldItems,
	}
}

func makeCompetitiveMon(uid string, slot int, monsNo uint16, nickname string, ability uint16, item uint16, nature byte, teraType uint32, shiny byte, evs [6]byte, moves []int16) model.ApiUserPokemonParam {
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
		B0:  uint32(100000 + slot),
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
		Bf:  []uint16{31, 31, 31, 31, 31, 31},
		Bg:  []int64{3, 3, 3, 3},
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
		Bv:  []int64{1},
		Bw:  0,
		Bx:  "Nicolas",
		By:  0,
		I:   uint32(slot),
	}
}

func defaultPlayerState() PlayerState {
	nowUnix := time.Now().Unix()
	now := uint64(nowUnix)
	pokes := map[string]model.ApiUserPokemonParam{
		"pk_001": makeCompetitiveMon("pk_001", 1, 6, "Charizard", 66, 660, 15, 2, 1, [6]byte{4, 0, 0, 252, 252, 0}, []int16{53, 403, 76, 182}),       // Charizard (Shiny, Charizardite Y)
		"pk_002": makeCompetitiveMon("pk_002", 2, 445, "Garchomp", 134, 270, 13, 5, 0, [6]byte{4, 252, 0, 252, 0, 0}, []int16{89, 337, 442, 182}),      // Garchomp (Rough Skin, Life Orb)
		"pk_003": makeCompetitiveMon("pk_003", 3, 448, "Lucario", 39, 675, 13, 2, 1, [6]byte{4, 252, 0, 252, 0, 0}, []int16{370, 418, 245, 14}),        // Lucario (Shiny, Lucarionite)
		"pk_004": makeCompetitiveMon("pk_004", 4, 94, "Gengar", 130, 275, 15, 8, 1, [6]byte{4, 0, 0, 252, 252, 0}, []int16{247, 188, 524, 182}),        // Gengar (Shiny, Focus Sash)
		"pk_005": makeCompetitiveMon("pk_005", 5, 149, "Dragonite", 136, 234, 3, 1, 0, [6]byte{252, 252, 0, 4, 0, 0}, []int16{245, 89, 7, 349}),        // Dragonite (Multiscale, Leftovers)
		"pk_006": makeCompetitiveMon("pk_006", 6, 376, "Metagross", 29, 759, 3, 9, 1, [6]byte{252, 252, 0, 4, 0, 0}, []int16{309, 418, 428, 8}),        // Metagross (Shiny, Metagrossite)
		"pk_007": makeCompetitiveMon("pk_007", 7, 25, "Pikachu", 31, 236, 15, 4, 0, [6]byte{4, 0, 0, 252, 252, 0}, []int16{85, 521, 98, 182}),          // Pikachu (Light Ball)
		"pk_008": makeCompetitiveMon("pk_008", 8, 282, "Gardevoir", 36, 657, 15, 18, 1, [6]byte{4, 0, 0, 252, 252, 0}, []int16{585, 94, 473, 182}),     // Gardevoir (Shiny, Gardevoirite)
		"pk_009": makeCompetitiveMon("pk_009", 9, 658, "Greninja", 168, 270, 15, 3, 1, [6]byte{4, 0, 0, 252, 252, 0}, []int16{56, 399, 58, 421}),       // Greninja (Shiny, Protean)
		"pk_010": makeCompetitiveMon("pk_010", 10, 248, "Tyranitar", 45, 640, 3, 6, 0, [6]byte{252, 252, 0, 0, 0, 4}, []int16{444, 242, 89, 182}),     // Tyranitar (Sand Stream, Assault Vest)
		"pk_011": makeCompetitiveMon("pk_011", 11, 373, "Salamence", 22, 769, 13, 10, 0, [6]byte{4, 252, 0, 252, 0, 0}, []int16{38, 89, 355, 349}),    // Salamence (Intimidate, Salamencite)
		"pk_012": makeCompetitiveMon("pk_012", 12, 681, "Aegislash", 176, 234, 17, 9, 1, [6]byte{252, 0, 0, 0, 252, 4}, []int16{247, 430, 425, 588}),  // Aegislash (Shiny, Stance Change)
	}

	teams := map[string]model.ApiUserTeamParam{
		"1": {
			Tid:    1,
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
		"2": {
			Tid:    2,
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
		UUID:         defaultUUID42,
		SupportID:    "SUP-CHAMP-9999",
		DeviceCd:     "DVC-CHAMP-0001",
		CsrfToken:    "tok_champ_private_0001",
		SessionID:    "sid_champ_private_0001",
		UdVer:        1,
		SchemaVer:    currentSchemaVer,
		RankSingle:   10,
		RatingSingle: 1500.0,
		WinsSingle:   10,
		LossesSingle: 0,
		RankDouble:   10,
		RatingDouble: 1500.0,
		WinsDouble:   10,
		LossesDouble: 0,
		SeasonPoints: 1000,
		Profile: model.ApiUserProfileParam{
			AgTA:   nowUnix,
			Bth:    200001,
			Clogc:  1,
			Cou:    31, // Brazil/Global
			CreAt:  nowUnix,
			Flng:   2,
			Fri:    "1234-5678-9012",
			Id:     defaultUUID42,
			Lavsp:  "1",
			Lavsw:  "1",
			Llday:  1,
			Lldaz:  1,
			Llog:   nowUnix,
			Lng:    2,
			Mode:   1,
			Motim:  nowUnix,
			Nam:    "Nicolas",
			PpV:    1,
			Push:   1,
			Sup:    "SUP-CHAMP-9999",
			Tlogc:  10,
			TmV:    1,
			Tzof:   -180,
			Dv:     "Android",
			Os:     "Android 14",
			Trtit:  "1",
			Trico:  "1",
			Trbad:  []string{"1"},
			TrnId:  "CHAMP-2026-0001",
			TrCAt:  now,
			TrIAt:  now,
			SwPNam: 1,
			SwBtDt: 1,
			SwPrDt: 1,
			AcFrRq: 1,
			HbuNam: 1,
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
			Subsc:  map[string]interface{}{"typ": 1, "fpt": now, "lst": now, "cst": 1, "exp": now + 31536000, "pmdid": 1},
			Misri: map[string]interface{}{
				"1": map[string]interface{}{"rd": "20261014", "mid": "1", "rrc": 3},
			},
			Emisi: map[string]interface{}{
				"1": map[string]interface{}{"end": nowUnix + 604800, "rrc": 3},
			},
			Putoi:  map[string]map[string]int{},
			Ptids:  []interface{}{},
			Mxrnk:  10,
			Nwnver: 1,
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
