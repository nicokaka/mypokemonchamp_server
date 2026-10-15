# MyPokeChampServer - Pokémon Champions 100% Private Server (v1.2.2-3859)

Servidor 100% privado, emulador de backend, painel web **MyPokeChamp Studio** e ferramental de engenharia reversa para **Pokémon Champions (`jp.pokemon.pokemonchampions` v1.2.2 Build 3859)**.

> 🤖 **Documentação Técnica & Engenharia Reversa:** Consulte [**`GUIA_PROXIMA_IA_DEPLOY_ORACLE.md`**](./GUIA_PROXIMA_IA_DEPLOY_ORACLE.md) para todos os detalhes de engenharia reversa (`libil2cpp.so`, `libunity.so`, `global-metadata.dat`, `classes.dex`), protocolo criptográfico `pmc`/`pmp` e arquitetura do **Smart Multiplexer** na Oracle Cloud (`164.152.40.114`).

---

## 🎯 Funcionalidades Implementadas

* **Smart TCP/TLS/HTTP Multiplexer (`cmd/server/main.go`):**
  * Compartilha a porta pública `443` da Oracle Cloud (`164.152.40.114`) sem interromper o servidor do **Pokémon TCG Pocket (`ptcgp.service` em `:8443`)**.
  * Atende conexões HTTP diretas do APK modificado (`http://164.152.40.114:443`), conexões HTTPS com SNI dinâmico (`*pokemonchampions*`) e encaminha demais conexões TLS via proxy TCP transparente para `127.0.0.1:8443`.
* **Criptografia Nativa Completa (`internal/crypto`):**
  * Implementação exata de `Net.Encrypt` (`AES-256-CBC` + `PKCS7` + `GZip` + derivação de chaves por `MD5` com `X-PKB-DMY-VAL`, `ServerNowUnixtime / 600` e `sessionId`/`csrfToken`/`udVer=1`).
  * Preenchimento automático de `resp.Pmc` em todas as respostas para compatibilidade total com `Net.ApiTemplate::ParseResponse` (`VA 0x2dd2b58`).
* **MyPokeChamp Studio (`/admin`):**
  * Painel web completo para editar Perfil de Treinador, Victory Points (VP), Ranking Single/Double, Box Pokémon (Espécie, Apelido, Shiny, Tera Type, Nature, Ability, Held Item, 6 IVs, 6 EVs e 4 Moves), Equipes de Batalha e Bots Campeões.
  * Botões rápidos para **Maximizar Conta (9.999.999 VP + Master Ball Rank)** e **Desbloquear Pacote Meta Competitivo**.
* **Matchmaking e Batalha via WebSocket (`internal/ws`):**
  * Suporte ao protocolo `WebSocket2` (`WSSendStartMatching` $\rightarrow$ `WSResponseMatchSuccess` $\rightarrow$ `WSSendCommand` $\rightarrow$ `WSResponseCommand`) com descriptografia de comandos `DecryptWSCommand` e IA de resposta de turno.
* **Gerador de APK Modificado (`scripts/gerar_apk_modificado.py`):**
  * Bypass da checagem de licença Google Play (`LicenseClient::checkLicense` em `classes.dex`).
  * Redirecionamento de `global-metadata.dat` (`StringLiteral[13153]`, `[10175]`, `[10176]`) para `http://164.152.40.114:443`.
  * Desbloqueio de HTTP em texto puro em `libunity.so` e assinatura automática dos 3 splits APK (`base`, `arm64_v8a`, `UnityDataAssetPack`).

---

## 📁 Estrutura do Repositório

```text
mypokechamp_server/
├── cmd/server/main.go                  # Entrypoint Smart Multiplexer (:7443) + HTTP/Admin (:9080)
├── game-data/master-data/*.json        # 89 tabelas reais de Master Data (5.033 registros extraídos do APK)
├── internal/
│   ├── api/router.go                   # Rotas /auth/*, /api/*, /health e API REST /api/admin/*
│   ├── api/md.go                       # Servidor criptografado de Master Data (/md/c/{sha1_hash})
│   ├── api/admin_ui.go                 # Interface Web MyPokeChamp Studio (/admin)
│   ├── api/router_test.go              # Testes automatizados de criptografia, boot, batalha e admin
│   ├── api/md_test.go                  # Teste ponta a ponta das 89 tabelas de Master Data
│   ├── crypto/crypto.go                # AES-256-CBC + GZip + derivação MD5 (Auth, API & WebSocket2)
│   ├── localtls/localtls.go            # Certificados TLS SNI dinâmicos
│   ├── model/dto.go                    # Structs canônicos extraídos do Net.dll
│   ├── store/store.go                  # Estado persistente JSON (Box, Times, VP, Ranking, Bots)
│   └── ws/battle_ws.go                 # Handler WebSocket2 para Matchmaking e Batalhas vs Bots
├── scripts/
│   ├── generate_master_data.py         # Extrator das 89 tabelas Master Data dos AssetBundles do APK
│   ├── deploy_champ_oracle.py          # Deploy automatizado via SSH/SCP para Oracle Cloud
│   └── gerar_apk_modificado.py         # Patcher de classes.dex, global-metadata.dat, libunity.so e libil2cpp.so
├── DEPLOY_SERVIDOR/
│   ├── instalar_champ_oracle.sh        # Instalador systemd (pokechamp.service)
│   ├── pokechamp-server-linux-arm64    # Binário Linux ARM64 (Oracle Ampere A1)
│   └── pokechamp-server-linux-amd64    # Binário Linux AMD64
├── instalar_no_celular.sh              # Script 1-clique para instalar os 3 splits APK via ADB
├── abrir_painel_oracle.sh              # Abre túnel SSH para o painel MyPokeChamp Studio da Oracle Cloud
├── GUIA_PROXIMA_IA_DEPLOY_ORACLE.md    # Documentação completa de Engenharia Reversa e Operação
├── ARQUITETURA_POKEMON_CHAMPIONS.md    # Catálogo de endpoints e protocolo
└── PLANO_MESTRE_POKEMON_CHAMPIONS.md   # Plano mestre do projeto
```

---

## 🚀 Como Usar

### 1. Acessar o Painel MyPokeChamp Studio (Oracle Cloud)
* Direto no navegador: `http://164.152.40.114:443/admin`
* Ou via túnel SSH local:
  ```bash
  ./abrir_painel_oracle.sh
  ```

### 2. Gerar e Instalar o APK Modificado no Celular (via USB / ADB)
```bash
python3 scripts/gerar_apk_modificado.py --url "http://164.152.40.114:443"
./instalar_no_celular.sh
```
