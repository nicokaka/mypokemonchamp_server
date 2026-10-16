package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/nicokaka/mypokechamp_server/internal/api"
	"github.com/nicokaka/mypokechamp_server/internal/localtls"
	"github.com/nicokaka/mypokechamp_server/internal/store"
)

type peekConn struct {
	net.Conn
	peeked []byte
}

func (c *peekConn) Read(b []byte) (int, error) {
	if len(c.peeked) > 0 {
		n := copy(b, c.peeked)
		c.peeked = c.peeked[n:]
		return n, nil
	}
	return c.Conn.Read(b)
}

type chanListener struct {
	addr net.Addr
	ch   chan net.Conn
	done chan struct{}
}

func (l *chanListener) Accept() (net.Conn, error) {
	select {
	case c, ok := <-l.ch:
		if !ok {
			return nil, net.ErrClosed
		}
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *chanListener) Close() error {
	select {
	case <-l.done:
	default:
		close(l.done)
	}
	return nil
}

func (l *chanListener) Addr() net.Addr {
	return l.addr
}

func isPTCGPTraffic(buf []byte) bool {
	// Any TLS traffic that is not specifically for Pokemon Champions belongs to PTCGP
	return !bytes.Contains(buf, []byte("pokemonchampions"))
}

func proxyTCP(client net.Conn, initial []byte, targetAddr string) {
	defer client.Close()
	backend, err := net.DialTimeout("tcp", targetAddr, 5*time.Second)
	if err != nil {
		log.Printf("[MUX] Erro ao conectar ao backend %s: %v", targetAddr, err)
		return
	}
	defer backend.Close()

	if len(initial) > 0 {
		if _, err := backend.Write(initial); err != nil {
			return
		}
	}

	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(backend, client)
		if tc, ok := backend.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(client, backend)
		if tc, ok := client.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
		done <- struct{}{}
	}()
	<-done
	<-done
}

func main() {
	addr := flag.String("addr", envOrDefault("CHAMP_ADDR", ":7443"), "Endereço/porta de escuta Multiplexada HTTPS/HTTP (ex: :7443)")
	httpAddr := flag.String("http-addr", envOrDefault("CHAMP_HTTP_ADDR", ":9080"), "Endereço/porta de escuta HTTP auxiliar (ex: :9080)")
	ptcgpBackend := flag.String("ptcgp-backend", envOrDefault("CHAMP_PTCGP_BACKEND", "127.0.0.1:8443"), "Backend ptcgp-server para coexistência na porta 443")
	dataDir := flag.String("data-dir", envOrDefault("CHAMP_DATA_DIR", "data"), "Diretório de persistência JSON")
	certDir := flag.String("cert-dir", envOrDefault("CHAMP_CERT_DIR", "certs"), "Diretório de certificados TLS/CA")
	publicHost := flag.String("public-host", envOrDefault("CHAMP_PUBLIC_HOST", "164.152.40.114"), "IP ou domínio público do servidor Oracle Cloud")
	flag.Parse()

	st, err := store.NewStore(*dataDir)
	if err != nil {
		log.Fatalf("Erro ao inicializar store: %v", err)
	}

	tlsCfg, err := localtls.DynamicTLSConfig(*certDir, []string{*publicHost})
	if err != nil {
		log.Fatalf("Erro ao preparar certificados TLS dinâmicos: %v", err)
	}

	publicURL := "https://" + *publicHost
	wsURL := "wss://" + *publicHost + "/ws"

	srv := api.NewServer(st, publicURL, wsURL)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	log.Printf("================================================================")
	log.Printf("  🏆 MyPokeChampServer - Pokémon Champions 100%% Private Server")
	log.Printf("  🔀 Smart Multiplexer   : %s (HTTP + HTTPS + PTCGP SNI Proxy)", *addr)
	log.Printf("  🌐 HTTP / Admin Panel  : http://localhost%s/admin", *httpAddr)
	log.Printf("  ☁️  Target Oracle Cloud : %s", publicURL)
	log.Printf("  🤖 WebSocket Match/Bot : %s", wsURL)
	log.Printf("================================================================")

	// Start auxiliary HTTP listener for quick local tests & Admin UI
	go func() {
		if err := http.ListenAndServe(*httpAddr, mux); err != nil {
			log.Printf("[HTTP] Listener auxiliar encerrado: %v", err)
		}
	}()

	rawLn, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("Erro fatal ao escutar em %s: %v", *addr, err)
	}

	cLn := &chanListener{
		addr: rawLn.Addr(),
		ch:   make(chan net.Conn, 256),
		done: make(chan struct{}),
	}

	go func() {
		for {
			conn, err := rawLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
				buf := make([]byte, 1024)
				n, err := c.Read(buf)
				_ = c.SetReadDeadline(time.Time{})
				if err != nil || n == 0 {
					_ = c.Close()
					return
				}
				initial := buf[:n]

				// TLS Handshake record starts with 0x16
				if initial[0] == 0x16 {
					if isPTCGPTraffic(initial) {
						proxyTCP(c, initial, *ptcgpBackend)
						return
					}
					pc := &peekConn{Conn: c, peeked: initial}
					tlsConn := tls.Server(pc, tlsCfg)
					cLn.ch <- tlsConn
					return
				}

				// Plaintext HTTP / WebSocket on the same port!
				pc := &peekConn{Conn: c, peeked: initial}
				cLn.ch <- pc
			}(conn)
		}
	}()

	httpServer := &http.Server{
		Handler: mux,
	}
	if err := httpServer.Serve(cLn); err != nil {
		log.Fatalf("Erro fatal no servidor HTTP/HTTPS: %v", err)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
