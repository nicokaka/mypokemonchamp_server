package ws

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/nicokaka/mypokechamp_server/internal/store"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type Handler struct {
	Store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{Store: s}
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

	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := rw.WriteString(resp); err != nil {
		return
	}
	if err := rw.Flush(); err != nil {
		return
	}

	log.Printf("[WS] Cliente conectado ao Matchmaking/Battle WebSocket (%s)", r.RemoteAddr)
	h.handleLoop(conn, rw)
}

func (h *Handler) handleLoop(conn net.Conn, rw *bufio.ReadWriter) {
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
			h.handleMessage(rw, payload)
		}
	}
}

func (h *Handler) handleMessage(rw *bufio.ReadWriter, payload []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(payload, &msg); err != nil {
		log.Printf("[WS] Frame não-JSON (%d bytes)", len(payload))
		return
	}
	action, _ := msg["action"].(string)
	cnt, _ := msg["cnt"].(float64)
	now := uint64(time.Now().Unix())
	log.Printf("[WS] Action recebida: %q (cnt=%v)", action, int(cnt))

	st := h.Store.GetState()
	bot := st.Bots[0]
	rmid := "rm_bot_champion_001"

	switch {
	case strings.Contains(strings.ToLower(action), "startmatching") || strings.Contains(strings.ToLower(action), "match"):
		// 1. Send MatchReady2
		readyResp := map[string]interface{}{
			"action": action,
			"code":   0,
			"lcode":  0,
			"cnt":    int(cnt),
			"stime":  now,
			"dp": map[string]interface{}{
				"rmid": rmid,
			},
		}
		sendJSON(rw, readyResp)

		// 2. Immediately send MatchSuccess with Bot opponent
		matchSuccess := map[string]interface{}{
			"action": "MatchSuccess",
			"code":   0,
			"lcode":  0,
			"cnt":    int(cnt) + 1,
			"stime":  now,
			"dp": map[string]interface{}{
				"bca":    1,
				"rul":    1,
				"stage":  1,
				"pkview": 1,
				"pkvtim": 60,
				"rmid":   rmid,
				"team": []interface{}{
					map[string]interface{}{
						"nam":    st.Profile.Nam,
						"trnId":  st.Profile.TrnId,
						"rat":    1500,
						"avatar": st.Avatar,
					},
					map[string]interface{}{
						"nam":    bot.TrainerName,
						"trnId":  "BOT-CHAMP-001",
						"rat":    bot.Ra,
						"avatar": st.Avatar,
						"mons":   bot.Mons,
					},
				},
			},
		}
		sendJSON(rw, matchSuccess)

	case strings.Contains(strings.ToLower(action), "pokemonselect") || strings.Contains(strings.ToLower(action), "readyok"):
		resp := map[string]interface{}{
			"action": action,
			"code":   0,
			"lcode":  0,
			"cnt":    int(cnt),
			"stime":  now,
			"dp": map[string]interface{}{
				"rmid":  rmid,
				"pssAt": now,
			},
		}
		sendJSON(rw, resp)

	case strings.Contains(strings.ToLower(action), "startbattle"):
		resp := map[string]interface{}{
			"action": action,
			"code":   0,
			"lcode":  0,
			"cnt":    int(cnt),
			"stime":  now,
			"dp": map[string]interface{}{
				"rmid":   rmid,
				"tse":    "bot_turn_seed_001",
				"tseasn": 1,
			},
		}
		sendJSON(rw, resp)

	case strings.Contains(strings.ToLower(action), "command"):
		dp, _ := msg["dp"].(map[string]interface{})
		bl := ""
		if dp != nil {
			bl, _ = dp["bl"].(string)
		}
		resp := map[string]interface{}{
			"action": action,
			"code":   0,
			"lcode":  0,
			"cnt":    int(cnt),
			"stime":  now,
			"dp": map[string]interface{}{
				"tobtim": 900,
				"comtim": 45,
				"btltim": 1200,
				"bl":     bl,
				"updAt":  now,
				"asn":    1,
			},
		}
		sendJSON(rw, resp)

	default:
		// KeepAlive or generic ack
		resp := map[string]interface{}{
			"action": action,
			"code":   0,
			"lcode":  0,
			"cnt":    int(cnt),
			"stime":  now,
			"dp": map[string]interface{}{
				"rmid": rmid,
			},
		}
		sendJSON(rw, resp)
	}
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
