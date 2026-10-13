package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/nicokaka/mypokechamp_server/internal/api"
	"github.com/nicokaka/mypokechamp_server/internal/localtls"
	"github.com/nicokaka/mypokechamp_server/internal/store"
)

func main() {
	addr := flag.String("addr", envOrDefault("CHAMP_ADDR", ":9443"), "Endereço/porta de escuta HTTPS (ex: :9443)")
	httpAddr := flag.String("http-addr", envOrDefault("CHAMP_HTTP_ADDR", ":9080"), "Endereço/porta de escuta HTTP auxiliar (ex: :9080)")
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

	publicURL := "https://" + *publicHost + *addr
	wsURL := "wss://" + *publicHost + *addr + "/ws"

	srv := api.NewServer(st, publicURL, wsURL)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	log.Printf("================================================================")
	log.Printf("  🏆 MyPokeChampServer - Pokémon Champions 100%% Private Server")
	log.Printf("  🔒 HTTPS (SNI Dinâmico): %s (Domínio: api.app.pokemonchampions.jp)", *addr)
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

	httpsServer := &http.Server{
		Addr:      *addr,
		Handler:   mux,
		TLSConfig: tlsCfg,
	}
	if err := httpsServer.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("Erro fatal no servidor HTTPS: %v", err)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
