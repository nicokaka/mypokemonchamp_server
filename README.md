# MyPokeChampServer - Pokémon Champions 100% Private Server (v1.2.2-3859)

Servidor 100% privado, emulador de backend e ferramental de engenharia reversa para **Pokémon Champions (`jp.pokemon.pokemonchampions` v1.2.2 Build 3859)**.

> 🤖 **Handoff para IA / Deploy Oracle Cloud:** Consulte o documento [**`GUIA_PROXIMA_IA_DEPLOY_ORACLE.md`**](./GUIA_PROXIMA_IA_DEPLOY_ORACLE.md) para todas as instruções de deploy via SSH na Oracle Cloud (`164.152.40.114`), offsets de engenharia reversa do `libil2cpp.so` / `global-metadata.dat` e assinatura do APK.

---

## 🎯 Funcionalidades Implementadas

* **Criptografia Nativa (`internal/crypto`):** Implementação completa do algoritmo `Net.Encrypt` (`AES-256-CBC` + `PKCS7` + `GZip` + derivação de chaves por `MD5` com `X-PKB-DMY-VAL`, `ServerNowUnixtime / 600` e `sessionId`/`csrfToken`/`udVer`) + suporte ao *fast-path* `pmp` de `DeserializeResponseBody`.
* **TLS SNI Dinâmico (`internal/localtls`):** Geração automática de CA raiz e certificados TLS assinados sob demanda para `api.app.pokemonchampions.jp` e `164.152.40.114`.
* **User Data (`UD`) & Store Persistente (`internal/store`):**
  * **999.999 VP (Victory Points)** liberados para treinamento gratuito;
  * **300 slots de Box** desbloqueados;
  * **12 Pokémon Competitivos Level 50** pré-carregados (*Charizard, Garchomp, Lucario, Gengar, Dragonite, Metagross, Pikachu, Gardevoir, Greninja, Tyranitar, Salamence, Aegislash*) com 6x31 IVs, EVs competitivos, Mega Stones e movimentos;
  * **5 Bots Campeões** (*Cynthia, Leon, Red, Steven, Blue*) configurados para batalhas.
* **Matchmaking e Batalha via WebSocket (`internal/ws`):** Suporte ao protocolo `WebSocket2` (`WSSendStartMatching` $\rightarrow$ `WSResponseMatchSuccess` $\rightarrow$ `WSSendCommand` $\rightarrow$ `WSResponseCommand`).
* **Painel Web Admin (`/admin`):** Interface web integrada na porta `9080` (HTTP) e `9443` (HTTPS) para visualizar a Box e injetar novos Pokémon competitivos em tempo real.

---

## 📁 Estrutura do Repositório

```text
mypokechamp_server/
├── cmd/server/main.go                  # Entrypoint HTTPS (:9443) + HTTP/Admin (:9080)
├── internal/
│   ├── api/router.go                   # Rotas /auth/*, /api/*, /health e Painel /admin
│   ├── api/router_test.go              # Testes automatizados de criptografia e fluxo completo
│   ├── crypto/crypto.go                # AES-256-CBC + GZip + derivação MD5 (Auth & API)
│   ├── localtls/localtls.go            # Certificados TLS SNI dinâmicos
│   ├── model/dto.go                    # Structs canônicos extraídos do Net.dll
│   ├── store/store.go                  # Estado persistente JSON (Box, Times, VP, Bots)
│   └── ws/battle_ws.go                 # Handler WebSocket2 para Matchmaking e Batalhas vs Bots
├── scripts/
│   ├── deploy_champ_oracle.py          # Deploy 1-clique via SSH/SCP para Oracle Cloud
│   └── gerar_apk_modificado.py         # Patcher de global-metadata.dat e libil2cpp.so
├── DEPLOY_SERVIDOR/
│   ├── instalar_champ_oracle.sh        # Instalador systemd (pokechamp.service na porta 9443)
│   ├── pokechamp-server-linux-arm64    # Binário Linux ARM64 (Oracle Ampere A1)
│   └── pokechamp-server-linux-amd64    # Binário Linux AMD64
├── GUIA_PROXIMA_IA_DEPLOY_ORACLE.md    # Manual completo para a próxima IA configurar via SSH
├── ARQUITETURA_POKEMON_CHAMPIONS.md    # Documentação da arquitetura
└── PLANO_MESTRE_POKEMON_CHAMPIONS.md   # Plano mestre do projeto
```

---

## 🚀 Como Executar e Fazer Deploy

### 1. Rodar Localmente
```bash
go run ./cmd/server
# Painel Admin: http://localhost:9080/admin
```

### 2. Deploy na Oracle Cloud (no PC com a chave `.key`)
```bash
python scripts/deploy_champ_oracle.py --key "CAMINHO/PARA/ssh-key-2026-09-19.key"
```

### 3. Gerar APK Modificado
```bash
python scripts/gerar_apk_modificado.py --url "https://164.152.40.114:9443" --skip-so-patch
```
