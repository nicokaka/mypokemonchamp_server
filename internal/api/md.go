package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/nicokaka/mypokechamp_server/internal/crypto"
	"github.com/nicokaka/mypokechamp_server/internal/model"
)

const (
	mdAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789_"
	mdListVer  = 1
	mdListTim  = 1710000018
	mdListRa   = "pkb2026md"
	mdListCry  = 1
)

type mdCachedEntry struct {
	Desc string
	Data []byte
}

var (
	mdCacheOnce    sync.Once
	mdFilesByHash  map[string]mdCachedEntry
	mdDefaultLSize int
	mdDefaultLChs  string
	mdDefaultLJhs  string
	cfTunnelRegex  = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)
)

var mdAllTableNames = []string{
	"country_codes",
	"months",
	"personal",
	"status_fo",
	"type",
	"tokusei",
	"seikaku",
	"reward_pokemon_params",
	"waza",
	"waza_category",
	"waza_target",
	"waza_learn",
	"waza_classification",
	"buffeffect",
	"item",
	"item_category",
	"ball",
	"ribbon_main",
	"ribbon_original",
	"training_cost",
	"regulation",
	"rule",
	"battle_pass_master",
	"battle_pass_reward",
	"battle_pass_random_reward",
	"display_item",
	"daily_mission_master",
	"daily_mission_condition",
	"daily_mission_step",
	"weekly_mission_master",
	"weekly_mission_condition",
	"weekly_mission_step",
	"event_mission_master",
	"event_mission_condition",
	"event_mission_step",
	"st_mission_master",
	"st_mission_condition",
	"st_mission_step",
	"subsc_mission_master",
	"subsc_mission_condition",
	"subsc_mission_step",
	"trainer_achievement_master",
	"trainer_achievement_condition",
	"trainer_achievement_step",
	"pokemon_achievement_master",
	"pokemon_achievement_condition",
	"pokemon_achievement_step",
	"shop_category",
	"shop_top",
	"paid_shop_item",
	"vp_shop_item",
	"battle_stage",
	"tab_costume",
	"tab_makeup",
	"costume",
	"avatar_color",
	"initial_avatar",
	"screen",
	"motion",
	"first_tutorial_event",
	"tutorial_flag",
	"tutorial_reward",
	"tutorial_rental",
	"bt_tutorial",
	"battle_status_tag",
	"omakase_assign",
	"omakase_mashi",
	"badge",
	"syougou",
	"trainer_icon",
	"profile_data",
	"user_resource_ticket",
	"btl_bgm_series",
	"btl_bgm",
	"club",
	"rental_common",
	"condition",
	"rank",
	"resource_type",
	"text_join",
	"championships",
	"ph_transfer_reward_condition",
	"pokelbl_shape",
	"pokelbl_color",
	"clsv_share_simple_params",
	"poke_regulation",
	"content_promotion",
	"bot_merge",
	"junior_ignore_resource",
}

func md5HexBytes(b []byte) string {
	return crypto.MD5Hex(string(b))
}

func sha256HexString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// encryptMdMetaListName is the exact inverse of Net.MdHandler::DecryptMdMetaListName (VA 0x453f5bc).
func encryptMdMetaListName(plainName string, tim int) string {
	n := len(mdAlphabet)
	shift := tim % n
	if shift < 0 {
		shift += n
	}
	var sb strings.Builder
	sb.Grow(len(plainName))
	for i := 0; i < len(plainName); i++ {
		idx := strings.IndexByte(mdAlphabet, plainName[i])
		if idx < 0 {
			sb.WriteByte(plainName[i])
		} else {
			encIdx := (idx - (shift % n) + n) % n
			sb.WriteByte(mdAlphabet[encIdx])
		}
		shift++
	}
	return sb.String()
}

// decryptMdMetaListName replicates Net.MdHandler::DecryptMdMetaListName (VA 0x453f5bc).
func decryptMdMetaListName(nse string, tim int) string {
	n := len(mdAlphabet)
	shift := tim % n
	if shift < 0 {
		shift += n
	}
	var sb strings.Builder
	sb.Grow(len(nse))
	for i := 0; i < len(nse); i++ {
		idx := strings.IndexByte(mdAlphabet, nse[i])
		if idx < 0 {
			sb.WriteByte(nse[i])
		} else {
			decIdx := (idx + shift) % n
			sb.WriteByte(mdAlphabet[decIdx])
		}
		shift++
	}
	return sb.String()
}

// deriveMdMetaListInfo replicates Net.MdHandler::GetMdMetaListInfo (VA 0x453f32c).
// Note: 0x453aa30 is EncryptUtils::SHA1 (40 hex chars).
func deriveMdMetaListInfo(platformCode, ver, tim int, ra string) (fileName, key, iv string) {
	fileName = crypto.SHA1Hex(fmt.Sprintf("%d%dc%d%s", platformCode, ver, tim, ra))
	key = crypto.MD5Hex(fmt.Sprintf("%d%da%d%s", platformCode, ver, tim, ra))
	ivFull := crypto.MD5Hex(fmt.Sprintf("%d%db%d%s", platformCode, ver, tim, ra))
	iv = ivFull[:16]
	return fileName, key, iv
}

// deriveMdMetaInfo replicates Net.MdHandler::GetMdMetaInfo (VA 0x453f96c).
// Note: 0x453fd58 calls 0x453aa30 (EncryptUtils::SHA1, 40 hex chars).
func deriveMdMetaInfo(platformCode int, name string, ver, tim int, ra string) (fileName, key, iv string) {
	timMd5 := crypto.MD5Hex(fmt.Sprintf("%d", tim))
	s1 := timMd5[0:10]
	s2 := timMd5[10:20]
	s3 := timMd5[20:30]

	h1 := crypto.MD5Hex(fmt.Sprintf("%d;%s;%s%d%s", ver, s1, name, platformCode, ra))
	h2 := crypto.MD5Hex(fmt.Sprintf("%s#%d#%s%d%s", name, ver, s2, platformCode, ra))
	h3 := crypto.MD5Hex(fmt.Sprintf("%s=%s=%d%d%s", s3, name, ver, platformCode, ra))

	fileName = crypto.SHA1Hex(fmt.Sprintf("%s%s%s%d", h3[0:10], h2[10:20], h1[20:30], ver))
	key = crypto.MD5Hex(fmt.Sprintf("%s@%s", name, h1))
	ivFull := crypto.MD5Hex(fmt.Sprintf("%s!%s", name, h2))
	iv = ivFull[:16]
	return fileName, key, iv
}

// encryptMdPayload encrypts UTF-8 JSON bytes using AES-256-CBC (UTF-8 key/iv) and encodes as Base64 ASCII bytes,
// matching Net.MdHandler::DecryptAndDeserialize<T> (VA 0x2c36094).
func encryptMdPayload(jsonStr, key, iv string) ([]byte, error) {
	cipherBytes, err := crypto.AesEncrypt([]byte(jsonStr), []byte(key), []byte(iv))
	if err != nil {
		return nil, err
	}
	b64 := base64.StdEncoding.EncodeToString(cipherBytes)
	return []byte(b64), nil
}

func buildMdTableJSON(tableName string) string {
	switch tableName {
	case "country_codes":
		rows := []map[string]interface{}{
			{
				"id":             1,
				"gid":            1,
				"ms_region_name": "option_syn",
				"ms_region_lbl":  "REGION_01",
				"gsort":          1,
				"code":           76,
				"bage":           13,
				"ms_name":        "option_syn",
				"ms_lbl":         "COUNTRY_076",
			},
			{
				"id":             2,
				"gid":            1,
				"ms_region_name": "option_syn",
				"ms_region_lbl":  "REGION_01",
				"gsort":          2,
				"code":           840,
				"bage":           13,
				"ms_name":        "option_syn",
				"ms_lbl":         "COUNTRY_840",
			},
			{
				"id":             3,
				"gid":            2,
				"ms_region_name": "option_syn",
				"ms_region_lbl":  "REGION_02",
				"gsort":          1,
				"code":           392,
				"bage":           13,
				"ms_name":        "option_syn",
				"ms_lbl":         "COUNTRY_392",
			},
		}
		b, _ := json.Marshal(rows)
		return string(b)

	case "months":
		rows := make([]map[string]interface{}, 0, 12)
		for m := 1; m <= 12; m++ {
			rows = append(rows, map[string]interface{}{
				"id":            m,
				"ms_name":       "option_syn",
				"ms_lbl":        fmt.Sprintf("MONTH_%02d", m),
				"ms_name_short": "option_syn",
				"ms_lbl_short":  fmt.Sprintf("MONTH_SHORT_%02d", m),
			})
		}
		b, _ := json.Marshal(rows)
		return string(b)

	default:
		return "[]"
	}
}

func initMdCache() {
	mdCacheOnce.Do(func() {
		mdFilesByHash = make(map[string]mdCachedEntry, len(mdAllTableNames)*4+4)

		for _, platformCode := range []int{0, 1, 2, 3} {
			metaEntries := make([]model.MdMeta, 0, len(mdAllTableNames))

			for _, name := range mdAllTableNames {
				tableJSON := buildMdTableJSON(name)
				fn, key, iv := deriveMdMetaInfo(platformCode, name, mdListVer, mdListTim, mdListRa)
				encBytes, err := encryptMdPayload(tableJSON, key, iv)
				if err != nil {
					log.Printf("[MD] erro ao criptografar tabela %s (plat=%d): %v", name, platformCode, err)
					continue
				}
				chs := md5HexBytes(encBytes)
				jhs := crypto.MD5Hex(tableJSON)
				nse := encryptMdMetaListName(name, mdListTim)

				mdFilesByHash[fn] = mdCachedEntry{
					Desc: fmt.Sprintf("table:%s (plat=%d)", name, platformCode),
					Data: encBytes,
				}

				metaEntries = append(metaEntries, model.MdMeta{
					Ver: mdListVer,
					Tim: mdListTim,
					Ra:  mdListRa,
					Chs: chs,
					Jhs: jhs,
					Siz: len(encBytes),
					Nse: nse,
				})
			}

			metaListObj := model.MdMetaList{List: metaEntries}
			metaListJSONBytes, _ := json.Marshal(metaListObj)
			metaListJSON := string(metaListJSONBytes)

			listFn, listKey, listIV := deriveMdMetaListInfo(platformCode, mdListVer, mdListTim, mdListRa)
			listEncBytes, err := encryptMdPayload(metaListJSON, listKey, listIV)
			if err != nil {
				log.Printf("[MD] erro ao criptografar MdMetaList (plat=%d): %v", platformCode, err)
				continue
			}

			entry := mdCachedEntry{
				Desc: fmt.Sprintf("MdMetaList (plat=%d, tables=%d)", platformCode, len(metaEntries)),
				Data: listEncBytes,
			}
			mdFilesByHash[listFn] = entry
			mdFilesByHash[sha256HexString(fmt.Sprintf("%d%dc%d%s", platformCode, mdListVer, mdListTim, mdListRa))] = entry

			// PlatformCode 3 is Android (RuntimePlatform.Android == 11 -> 3).
			if platformCode == 3 {
				mdDefaultLSize = len(listEncBytes)
				mdDefaultLChs = md5HexBytes(listEncBytes)
				mdDefaultLJhs = crypto.MD5Hex(metaListJSON)
			}
		}
	})
}

func resolveMdBaseURL(s *Server, r *http.Request) string {
	if envURL := strings.TrimSpace(os.Getenv("POKECHAMP_MD_URL")); envURL != "" {
		return strings.TrimRight(envURL, "/")
	}
	if data, err := os.ReadFile("/tmp/pokechamp-tunnel.log"); err == nil && len(data) > 0 {
		matches := cfTunnelRegex.FindAllString(string(data), -1)
		if len(matches) > 0 {
			return strings.TrimRight(matches[len(matches)-1], "/")
		}
	}
	return strings.TrimRight(s.resolvePublicURL(r), "/")
}

func (s *Server) buildMdListMeta(r *http.Request) *model.MdListMeta {
	initMdCache()
	dom := resolveMdBaseURL(s, r) + "/md/"
	return &model.MdListMeta{
		Ver:  mdListVer,
		Tim:  mdListTim,
		Ra:   mdListRa,
		Cry:  mdListCry,
		Dom:  dom,
		Ljhs: mdDefaultLJhs,
		Lchs: mdDefaultLChs,
		Lsiz: mdDefaultLSize,
	}
}

func (s *Server) handleMasterData(w http.ResponseWriter, r *http.Request) {
	initMdCache()

	// Client requests: GET /md/c/{fileName} (or /md/{fileName})
	trimmed := strings.TrimPrefix(r.URL.Path, "/md/")
	trimmed = strings.TrimPrefix(trimmed, "c/")
	fileName := strings.Trim(trimmed, "/")

	entry, ok := mdFilesByHash[fileName]
	if !ok {
		log.Printf("[MD] 404 NOT FOUND: %s %s from %s (fileName=%s)", r.Method, r.URL.Path, r.RemoteAddr, fileName)
		http.NotFound(w, r)
		return
	}

	log.Printf("[MD] 200 OK: %s %s -> %s (%d bytes)", r.Method, r.URL.Path, entry.Desc, len(entry.Data))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(entry.Data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(entry.Data)
}
