package ws

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/nicokaka/mypokechamp_server/internal/crypto"
	"github.com/nicokaka/mypokechamp_server/internal/model"
	"github.com/nicokaka/mypokechamp_server/internal/store"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type Handler struct {
	Store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{Store: s}
}

type sessionState struct {
	seqNo    uint16
	asn      int
	hasEntry bool
}

func packUnit(userIdx, pokeIdx, side, pos int) int16 {
	u := (userIdx & 0xF) | ((pokeIdx & 0xF) << 4) | ((side & 0xF) << 8) | ((pos & 0xF) << 12)
	return int16(uint16(u))
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ws_endpoint_ready",
			"message": "Connect via WebSocket2 client for Bot Matchmaking & Battles",
		})
		return
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	conn, rw, err := hj.Hijack()
	if err != nil {
		log.Printf("[WS] Hijack error: %v", err)
		return
	}
	defer conn.Close()

	sum := sha1.Sum([]byte(key + wsGUID))
	accept := base64.StdEncoding.EncodeToString(sum[:])

	protoHdr := ""
	if rawProto := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Protocol")); rawProto != "" {
		firstProto := strings.TrimSpace(strings.Split(rawProto, ",")[0])
		if firstProto != "" {
			protoHdr = "Sec-WebSocket-Protocol: " + firstProto + "\r\n"
		}
	}

	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n" +
		protoHdr + "\r\n"
	if _, err := rw.WriteString(resp); err != nil {
		return
	}
	if err := rw.Flush(); err != nil {
		return
	}

	log.Printf("[WS] Cliente conectado ao Matchmaking/Battle WebSocket2 (%s)", r.RemoteAddr)
	sess := &sessionState{}
	h.handleLoop(conn, rw, sess)
}

func (h *Handler) handleLoop(conn net.Conn, rw *bufio.ReadWriter, sess *sessionState) {
	for {
		opcode, payload, err := readFrame(rw)
		if err != nil {
			if err != io.EOF {
				log.Printf("[WS] Conexão encerrada: %v", err)
			}
			return
		}
		switch opcode {
		case 0x8: // Close
			_ = writeFrame(rw, 0x8, nil)
			return
		case 0x9: // Ping
			_ = writeFrame(rw, 0xA, payload)
		case 0x1, 0x2: // Text / Binary
			h.handleMessage(rw, payload, sess)
		}
	}
}

func cntAsString(v interface{}) string {
	switch val := v.(type) {
	case string:
		if val != "" {
			return val
		}
		return "1"
	case float64:
		return fmt.Sprintf("%d", int64(val))
	case int:
		return fmt.Sprintf("%d", val)
	default:
		return "1"
	}
}

func (h *Handler) handleMessage(rw *bufio.ReadWriter, payload []byte, sess *sessionState) {
	var msg map[string]interface{}
	if err := json.Unmarshal(payload, &msg); err != nil {
		log.Printf("[WS] Frame não-JSON (%d bytes)", len(payload))
		return
	}
	action, _ := msg["action"].(string)
	cntStr := cntAsString(msg["cnt"])
	dmy, _ := msg["dmy"].(string)
	now := time.Now().Unix()
	if dmy == "" {
		dmy = fmt.Sprintf("%d", now)
	}
	log.Printf("[WS] Action recebida: %q (cnt=%s)", action, cntStr)

	st := h.Store.GetState()
	rmid := "rm_bot_champion_001"
	actLower := strings.ToLower(action)

	switch {
	case actLower == "send_start_matching" || strings.Contains(actLower, "startmatching") || strings.Contains(actLower, "start_matching"):
		// 1. Acknowledge start matching (WebSocket2.WSAction.CL.RES_START_SUCCESS)
		sendJSON(rw, map[string]interface{}{
			"action": "res_start_matching",
			"code":   0,
			"lcode":  0,
			"cnt":    cntStr,
			"spt":    now,
			"stime":  now,
			"dmy":    dmy,
		})

		// 2. Notify match ready (WebSocket2.WSAction.CL.SEND_MATCH_READY2)
		sendJSON(rw, map[string]interface{}{
			"action": "send_match_ready2",
			"code":   0,
			"lcode":  0,
			"cnt":    cntStr,
			"spt":    now,
			"stime":  now,
			"dmy":    dmy,
			"dp": map[string]interface{}{
				"rmid": rmid,
			},
		})

		// 3. Also send MatchSuccess immediately with full player + bot teams (Net.ApiTeamParam[])
		playerTeam := h.Store.BuildPlayerBattleTeam()
		botTeam := h.Store.BuildBotBattleTeam(0)
		sendJSON(rw, map[string]interface{}{
			"action": "send_match_success",
			"code":   0,
			"lcode":  0,
			"cnt":    cntStr,
			"spt":    now,
			"stime":  now,
			"dmy":    dmy,
			"dp": map[string]interface{}{
				"bca":    byte(1),
				"rul":    byte(1),
				"stage":  1,
				"pkview": 1,
				"pkvtim": 60,
				"rmid":   rmid,
				"team":   []model.ApiTeamParam{playerTeam, botTeam},
			},
			"dg": "",
			"dc": "",
		})

	case actLower == "send_ready_ok" || strings.Contains(actLower, "readyok") || strings.Contains(actLower, "ready_ok"):
		playerTeam := h.Store.BuildPlayerBattleTeam()
		botTeam := h.Store.BuildBotBattleTeam(0)
		sendJSON(rw, map[string]interface{}{
			"action": "send_match_success",
			"code":   0,
			"lcode":  0,
			"cnt":    cntStr,
			"spt":    now,
			"stime":  now,
			"dmy":    dmy,
			"dp": map[string]interface{}{
				"bca":    byte(1),
				"rul":    byte(1),
				"stage":  1,
				"pkview": 1,
				"pkvtim": 60,
				"rmid":   rmid,
				"team":   []model.ApiTeamParam{playerTeam, botTeam},
			},
			"dg": "",
			"dc": "",
		})

	case actLower == "send_ready_pokemon_select" || strings.Contains(actLower, "pokemonselect") || strings.Contains(actLower, "pokemon_select"):
		sendJSON(rw, map[string]interface{}{
			"action": "res_ready_pokemon_select",
			"code":   0,
			"lcode":  0,
			"cnt":    cntStr,
			"spt":    now,
			"stime":  now,
			"dmy":    dmy,
		})
		sendJSON(rw, map[string]interface{}{
			"action": "send_pokemon_select_start",
			"code":   0,
			"lcode":  0,
			"cnt":    cntStr,
			"spt":    now,
			"stime":  now,
			"dmy":    dmy,
			"dp": map[string]interface{}{
				"pssAt": now,
			},
		})

	case actLower == "send_start_battle" || actLower == "send_start_battle_timeout" || strings.Contains(actLower, "startbattle") || strings.Contains(actLower, "start_battle"):
		sendJSON(rw, map[string]interface{}{
			"action": "res_start_battle",
			"code":   0,
			"lcode":  0,
			"cnt":    cntStr,
			"spt":    now,
			"stime":  now,
			"dmy":    dmy,
		})
		sendJSON(rw, map[string]interface{}{
			"action": "send_start_battle",
			"code":   0,
			"lcode":  0,
			"cnt":    cntStr,
			"spt":    now,
			"stime":  now,
			"dmy":    dmy,
			"dp": map[string]interface{}{
				"fri": st.Profile.Fri,
			},
		})
		sendJSON(rw, map[string]interface{}{
			"action": "send_battle_start",
			"code":   0,
			"lcode":  0,
			"cnt":    cntStr,
			"spt":    now,
			"stime":  now,
			"dmy":    dmy,
			"dp": map[string]interface{}{
				"tse":    []int{0, 1, 2, 3},
				"tseasn": 1,
			},
		})

	case actLower == "command" || strings.Contains(actLower, "command"):
		sub, _ := msg["sub"].(string)
		if sub == "" {
			sub = "command"
		}
		dp, _ := msg["dp"].(map[string]interface{})
		if dp == nil {
			if dcStr, ok := msg["dc"].(string); ok && dcStr != "" {
				if dec, err := crypto.DecryptWSCommand(dcStr, st.SessionID, "atk_bot_match_001", dmy); err == nil {
					_ = json.Unmarshal([]byte(dec), &dp)
				}
			}
		}

		inBl := ""
		if dp != nil {
			inBl, _ = dp["bl"].(string)
		}

		outBl, endType := h.buildBattleTurnResponse(inBl, sess)
		sess.asn++

		resp := map[string]interface{}{
			"action": "res_command",
			"code":   0,
			"lcode":  0,
			"cnt":    cntStr,
			"spt":    now,
			"stime":  now,
			"dmy":    dmy,
			"sub":    sub,
			"cry":    0,
			"hsh":    "",
			"dc":     "",
			"dp": map[string]interface{}{
				"tobtim": 900,
				"comtim": 45,
				"btltim": map[string]interface{}{
					"sc":     map[string]int{"0": 1, "1": 1},
					"btlst":  now,
					"us":     []map[string]interface{}{{"p": now, "f": 0, "s": 0}, {"p": now, "f": 0, "s": 0}},
					"effst":  now,
					"effmax": 60,
					"keffst": 0,
					"kast":   0,
					"cmdst":  now,
				},
				"bl":    outBl,
				"et":    endType,
				"lfri":  []string{},
				"updAt": now,
				"asn":   sess.asn,
			},
		}
		sendJSON(rw, resp)

	case actLower == "keepalive":
		sendJSON(rw, map[string]interface{}{
			"action": "keepalive",
			"code":   0,
			"lcode":  0,
			"cnt":    cntStr,
			"spt":    now,
			"stime":  now,
			"dmy":    dmy,
		})

	default:
		sendJSON(rw, map[string]interface{}{
			"action": action,
			"code":   0,
			"lcode":  0,
			"cnt":    cntStr,
			"spt":    now,
			"stime":  now,
			"dmy":    dmy,
			"dp": map[string]interface{}{
				"rmid": rmid,
			},
		})
	}
}

// buildBattleTurnResponse constructs a valid WebSocket.Message.Receive.BattleCommandResponse JSON string ("Sequences")
// matching PKB.Battle.BattleSequencePlayer & BattleSequenceUtil (Types 13370, 13371, 13375).
func (h *Handler) buildBattleTurnResponse(inBl string, sess *sessionState) (string, int) {
	// Parse incoming BattleCommand if provided
	moveID := int16(89) // Default: Earthquake
	if inBl != "" {
		var cmdMap map[string]interface{}
		if err := json.Unmarshal([]byte(inBl), &cmdMap); err == nil {
			if m, ok := cmdMap["Move"].(float64); ok && m > 0 {
				moveID = int16(m)
			}
		}
	}

	nextNo := func() uint16 {
		sess.seqNo++
		return sess.seqNo
	}

	p0 := packUnit(0, 0, 0, 0)
	p1 := packUnit(1, 0, 1, 0)

	// First command in battle: send CtrlEntry (2) for both sides + SyncStartTurn (49) + SyncEndCmd (59) + CtrlWait (1)
	if !sess.hasEntry {
		sess.hasEntry = true
		seqs := []map[string]interface{}{
			{"Cmd": 2, "SU": p0, "TU": p0, "No": nextNo()},  // CtrlEntry Player
			{"Cmd": 2, "SU": p1, "TU": p1, "No": nextNo()},  // CtrlEntry Bot
			{"Cmd": 49, "P1": 1, "No": nextNo()},            // SyncStartTurn
			{"Cmd": 59, "No": nextNo()},                     // SyncEndCmd
			{"Cmd": 1, "No": nextNo()},                      // CtrlWait
		}
		b, _ := json.Marshal(map[string]interface{}{"Sequences": seqs})
		return string(b), 0
	}

	// Subsequent turn command: execute player move, KO bot Pokemon, and trigger CtrlBattleEnd (3)
	seqs := []map[string]interface{}{
		{"Cmd": 52, "SU": p0, "TU": p1, "No": nextNo()},                               // SyncExeActBegin
		{"Cmd": 14, "SU": p0, "TU": p1, "EKind": 1, "EMdId": moveID, "No": nextNo()},  // CtrlActMoveSuccess
		{"Cmd": 38, "SU": p0, "TU": p1, "P1": 250, "No": nextNo()},                    // CtrlHPDamage
		{"Cmd": 4, "SU": p1, "TU": p1, "No": nextNo()},                                // CtrlPokeFainting
		{"Cmd": 55, "SU": p0, "TU": p1, "No": nextNo()},                               // SyncExeActEnd
		{"Cmd": 58, "P1": 1, "No": nextNo()},                                          // SyncEndTurn
		{"Cmd": 3, "P1": 1, "RU": 0, "No": nextNo()},                                  // CtrlBattleEnd (Player Win)
		{"Cmd": 59, "No": nextNo()},                                                   // SyncEndCmd
		{"Cmd": 1, "No": nextNo()},                                                    // CtrlWait
	}
	b, _ := json.Marshal(map[string]interface{}{"Sequences": seqs})
	return string(b), 1
}

func sendJSON(rw *bufio.ReadWriter, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	_ = writeFrame(rw, 0x1, b)
}

func readFrame(rw *bufio.ReadWriter) (byte, []byte, error) {
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(rw, hdr); err != nil {
		return 0, nil, err
	}
	opcode := hdr[0] & 0x0f
	masked := (hdr[1] & 0x80) != 0
	length := uint64(hdr[1] & 0x7f)

	switch length {
	case 126:
		ext := make([]byte, 2)
		if _, err := io.ReadFull(rw, ext); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext))
	case 127:
		ext := make([]byte, 8)
		if _, err := io.ReadFull(rw, ext); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(ext)
	}

	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(rw, maskKey[:]); err != nil {
			return 0, nil, err
		}
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(rw, payload); err != nil {
		return 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}
	return opcode, payload, nil
}

func writeFrame(rw *bufio.ReadWriter, opcode byte, payload []byte) error {
	var hdr []byte
	hdr = append(hdr, 0x80|opcode)
	n := len(payload)
	if n < 126 {
		hdr = append(hdr, byte(n))
	} else if n <= 65535 {
		hdr = append(hdr, 126, byte(n>>8), byte(n))
	} else {
		hdr = append(hdr, 127,
			byte(uint64(n)>>56), byte(uint64(n)>>48), byte(uint64(n)>>40), byte(uint64(n)>>32),
			byte(uint64(n)>>24), byte(uint64(n)>>16), byte(uint64(n)>>8), byte(uint64(n)))
	}
	if _, err := rw.Write(hdr); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := rw.Write(payload); err != nil {
			return err
		}
	}
	return rw.Flush()
}
