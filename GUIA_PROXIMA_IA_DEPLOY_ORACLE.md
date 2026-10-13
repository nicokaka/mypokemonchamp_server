# 🤖 Guia de Handoff para a Próxima IA — Deploy Oracle Cloud & APK Pokémon Champions

> **Destinatário:** Próximo agente de IA (ou desenvolvedor) executando no computador principal do usuário que possui a chave SSH da Oracle Cloud (`ssh-key-2026-09-19.key`).
> **Status Atual:** Servidor 100% codificado em Go, testado (`go test ./...` PASS), binários Linux (`arm64` e `amd64`) pré-compilados em `DEPLOY_SERVIDOR/` e scripts de automação prontos.

---

## 1. Contexto Geral e Decisão de Arquitetura

1. **Escopo Definido pelo Usuário:**
   - Construção de um **Servidor 100% Privado (`mypokechamp_server`)** para **Pokémon Champions (`jp.pokemon.pokemonchampions` v1.2.2 Build 3859)**.
   - Integração com Pokémon HOME oficial e matchmaking online oficial foram descartados por serem validados server-to-server nos servidores da Nintendo/ILCA. Todo o ecossistema (Box, Times, Treinamento com VP infinito, Matchmaking e Batalhas contra Bots via WebSocket) roda **100% dentro deste servidor privado**.
2. **Coexistência na Mesma VM Oracle Cloud (`164.152.40.114`):**
   - A VM Oracle Cloud Free Tier do usuário (`ubuntu@164.152.40.114`) **já roda o servidor privado de Pokémon TCG Pocket (`ptcgp.service` em `/opt/ptcgp-server`)** ocupando as portas `443 -> 8443` e `8081`.
   - **REGRA DE OURO PARA A PRÓXIMA IA:** **NUNCA** pare, altere ou sobrescreva `/opt/ptcgp-server`, o serviço `ptcgp.service` ou o redirecionamento `iptables` da porta `443 -> 8443`.
   - O **MyPokeChampServer (`pokechamp.service`)** foi projetado para rodar isoladamente em **`/opt/pokechamp-server`** usando:
     - **Porta `9443` (TCP):** Servidor Principal HTTPS (com geração dinâmica de certificados TLS por SNI) + WebSocket (`wss://164.152.40.114:9443/ws`) para Matchmaking e Batalhas contra Bots.
     - **Porta `9080` (TCP):** Listener HTTP auxiliar e Painel Web Admin (`http://164.152.40.114:9080/admin`).

---

## 2. Descobertas da Engenharia Reversa (`v1.2.2 Build 3859`)

Caso precise ajustar qualquer detalhe do protocolo ou do patcher do APK, aqui estão os mapeamentos exatos extraídos de `global-metadata.dat` (IL2CPP v31, magic `0xFAB11BAF`) e `libil2cpp.so` (ARM64):

### 2.1. Mapeamento ELF de `libil2cpp.so` (`va_to_off`)
- `PT_LOAD [1]`: `vaddr = 0x0 .. 0x1ed94b8` $\rightarrow$ `file_offset = va`
- `PT_LOAD [2]`: `vaddr = 0x1edd4c0 .. 0x5096b20` $\rightarrow$ `file_offset = va - 0x4000`
- `PT_LOAD [3]`: `vaddr = 0x509ab20 .. 0x53ca588` $\rightarrow$ `file_offset = va - 0x8000`
- `PT_LOAD [4]`: `vaddr = 0x53ce588 .. 0x56be808` $\rightarrow$ `file_offset = va - 0xc000`

### 2.2. Endereços Virtuais (VAs) Críticos em `libil2cpp.so` (`Net.dll`)
- `Net.ApiBase::CreateRequestBodyBase`: `0x451d120`
  - `0x451d1e0`: `BL 0x4537950` (`Net.Encrypt::EncryptRequestPmpAuth`)
  - `0x451d23c`: `BL 0x45386c0` (`Net.Encrypt::EncryptRequestPmpApi`)
- `Net.ApiBase::CreateHeader`: `0x451d41c` (Injeta `X-PKB-ENCRYPT`, `X-PKB-HASH-CHECK`, `X-PKB-DMY-VAL`)
- `Net.ApiBase::<Response>d__39::MoveNext`: `0x45200dc` (Lê `X-PKB-S-TIME`, `X-PKB-ENCRYPT`, `X-PKB-DMY-VAL`)
- `Net.ApiBase::CheckResponseBodyBase`: `0x451d9d8`
- `Net.ApiRegisteredTemplate::CreateRequestBody`: `0x2dd2890`
- `Net.ApiRegisteredTemplate::DeserializeResponseBody`: `0x2dd2b58`
  - **Fast-path nativo em `0x2dd2c38`:** O cliente desserializa o JSON completo (`TResponseBody`) em `0x2dd2c24` e verifica `String.IsNullOrEmpty(response.pmc)` em `0x2dd2c38`. Se `pmc` estiver vazio/omitido e `pmp` estiver presente no JSON (`{"code": 0, "pmp": {...}}`), o cliente usa `response.pmp` diretamente e retorna `ApiErrorType.None (0)` sem exigir descriptografia!
- `Net.Encrypt::EncryptRequestPmpAuth`: `0x4537950`
- `Net.Encrypt::EncryptRequestPmpApi`: `0x45386c0`
- `Net.Encrypt::DecryptResponsePmcAuth`: `0x4538b48`
- `Net.Encrypt::DecryptResponsePmcApi`: `0x4539900`
- `Net.NetEnvironment::SetApiEndpoint`: `0x453aff4`
- `Net.NetEnvironment::GetApiEndPoint`: `0x453b3f4`
- `Net.NetEnvironment::.cctor`: `0x453b528`

### 2.3. Algoritmo de Criptografia `pmc` Implementado (`internal/crypto/crypto.go`)
- **Compressão + Cifra:** `GZip` seguido de `AES-256-CBC` com padding `PKCS7` e codificado em `Base64`.
- **Derivação de Chave/IV para `/auth/*` (`EncryptPmpAuth` / `DecryptPmcAuth`):**
  - `win = uint32(serverNowUnixtime / 600)` (no primeiro request antes de receber `X-PKB-S-TIME`, o cliente pode usar `win = 0`; nosso servidor testa `nowWin`, `0`, `nowWin-1` e `nowWin+1` automaticamente).
  - `md5Dummy = MD5Hex(dummyValue)` (onde `dummyValue` vem do header HTTP `X-PKB-DMY-VAL`).
  - `h2 = MD5Hex(fmt.Sprintf("%d$%s", win, md5Dummy))`
  - `combined = md5Dummy + h2` (64 caracteres hex)
  - `shift = int(win & 31)`
  - `key = HexDecode(combined[shift:] + combined[:shift])` (32 bytes = 256 bits)
  - `iv = HexDecode(MD5Hex(fmt.Sprintf("%s;%d", dummyValue, win)))` (16 bytes = 128 bits)
- **Derivação de Chave/IV para `/api/*` (`EncryptPmpApi` / `DecryptPmcApi`):**
  - `h1 = MD5Hex(fmt.Sprintf("%s$%s$%s", sessionId, csrfToken, dummyValue))`
  - `shift = int(ParseInt64(dummyValue) & 31)`
  - `rotated = h1[shift:] + h1[:shift]`
  - `h2 = MD5Hex(fmt.Sprintf("%d%s", udVer, rotated))`
  - `key = HexDecode(h1 + h2)` (32 bytes = 256 bits)
  - `iv = HexDecode(MD5Hex(fmt.Sprintf("%s=%s=%s", sessionId, dummyValue, csrfToken)))` (16 bytes = 128 bits)

### 2.4. Strings Críticas em `global-metadata.dat`
- `StringLiteral[13153]`: `'https://api.app.pokemonchampions.jp'` (35 bytes) — Endpoint inicial do jogo.
- `StringLiteral[10175]`: `'^https://([0-9A-Za-z\\-]+)?api(\\.[0-9A-Za-z\\-]+)?\\.app\\.pokemonchampions\\.jp/?$'` (78 bytes) — Regex que valida as URLs recebidas em `envLs` no `/auth/get-token`.
- `StringLiteral[10176]`: `'^https://([0-9A-Za-z]+)\\-api'` (28 bytes) — Segundo Regex de ambiente em `SetApiEndpoint`.

---

## 3. Arquitetura do Repositório

| Arquivo | Função |
| :--- | :--- |
| [`cmd/server/main.go`](file:///c:/Users/nicolas/.gemini/antigravity-ide/scratch/mypokechamp_server/cmd/server/main.go) | Entrypoint do servidor; inicia HTTPS com SNI dinâmico (`:9443`) e HTTP auxiliar (`:9080`). |
| [`internal/crypto/crypto.go`](file:///c:/Users/nicolas/.gemini/antigravity-ide/scratch/mypokechamp_server/internal/crypto/crypto.go) | Criptografia/descriptografia real `AES-256-CBC` + `GZip` + `MD5` para `/auth/*` e `/api/*`. |
| [`internal/localtls/localtls.go`](file:///c:/Users/nicolas/.gemini/antigravity-ide/scratch/mypokechamp_server/internal/localtls/localtls.go) | Geração automática de CA raiz e certificados TLS folha dinâmicos por SNI (`api.app.pokemonchampions.jp`, `164.152.40.114`). |
| [`internal/model/dto.go`](file:///c:/Users/nicolas/.gemini/antigravity-ide/scratch/mypokechamp_server/internal/model/dto.go) | Structs canônicos do `Net.dll` (`ApiRequestEnvelope`, `ApiResponseEnvelope`, `ApiUdParam`, `ApiUserPokemonParam`, etc.). |
| [`internal/store/store.go`](file:///c:/Users/nicolas/.gemini/antigravity-ide/scratch/mypokechamp_server/internal/store/store.go) | Persistência em `data/player_state.json` com 999.999 VP, 12 Pokémon competitivos Shiny/Mega Lv.50, 2 times e 5 Bots Campeões. |
| [`internal/api/router.go`](file:///c:/Users/nicolas/.gemini/antigravity-ide/scratch/mypokechamp_server/internal/api/router.go) | Handlers de todas as rotas `/auth/*`, `/api/*`, `/health` e Painel Web Admin (`/admin`). |
| [`internal/ws/battle_ws.go`](file:///c:/Users/nicolas/.gemini/antigravity-ide/scratch/mypokechamp_server/internal/ws/battle_ws.go) | Servidor WebSocket RFC 6455 nativo compatível com `WebSocket2` para Matchmaking e Batalhas contra Bots. |
| [`scripts/deploy_champ_oracle.py`](file:///c:/Users/nicolas/.gemini/antigravity-ide/scratch/mypokechamp_server/scripts/deploy_champ_oracle.py) | Compila os binários Linux e faz o deploy completo via SSH/SCP para `ubuntu@164.152.40.114`. |
| [`DEPLOY_SERVIDOR/instalar_champ_oracle.sh`](file:///c:/Users/nicolas/.gemini/antigravity-ide/scratch/mypokechamp_server/DEPLOY_SERVIDOR/instalar_champ_oracle.sh) | Script executado na VM Oracle Cloud: cria `/opt/pokechamp-server`, abre portas `9443` e `9080` no `iptables` e ativa `pokechamp.service`. |
| [`scripts/gerar_apk_modificado.py`](file:///c:/Users/nicolas/.gemini/antigravity-ide/scratch/mypokechamp_server/scripts/gerar_apk_modificado.py) | Aplica patch em `global-metadata.dat` (e opcionalmente em `libil2cpp.so`) dentro do APK do Pokémon Champions. |

---

## 4. Passo a Passo para a Próxima IA (Deploy na Oracle Cloud)

Quando o usuário abrir este projeto no computador que contém a chave SSH (`ssh-key-2026-09-19.key`), siga exatamente estes passos:

### Passo 1: Executar o Deploy Automatizado via SSH
Localize o arquivo `.key` (geralmente em `~/.ssh/ssh-key-2026-09-19.key`, `Downloads` ou na pasta do `mytcgserver`) e execute:

```powershell
python scripts/deploy_champ_oracle.py --key "CAMINHO_COMPLETO_DA_CHAVE.key"
```

O script irá automaticamente:
1. Compilar `pokechamp-server-linux-arm64` e `pokechamp-server-linux-amd64`.
2. Enviar os binários e `instalar_champ_oracle.sh` para `/tmp/pokechamp_deploy` em `ubuntu@164.152.40.114`.
3. Instalar o serviço `pokechamp.service` em `/opt/pokechamp-server`, liberar as portas `9443` e `9080` no `iptables` da VM (sem afetar o `ptcgp.service` na `8443`) e iniciar o servidor.

### Passo 2: Liberar a Regra de Entrada (Ingress Rule) no Painel da Oracle Cloud (VCN)
Se o teste externo em `https://164.152.40.114:9443/health` ou `http://164.152.40.114:9080/admin` der timeout:
- Lembre o usuário de adicionar no painel web da **Oracle Cloud -> Networking -> Virtual Cloud Networks -> Security Lists -> Ingress Rules**:
  - **Source CIDR:** `0.0.0.0/0`
  - **IP Protocol:** `TCP`
  - **Destination Port Range:** `9443` e `9080`

### Passo 3: Verificar o Servidor e o Painel Admin
Teste a resposta da VM:
```powershell
curl.exe -k https://164.152.40.114:9443/health
```
E acesse o painel de gerenciamento da Box Pokémon pelo navegador:
- `http://164.152.40.114:9080/admin`

### Passo 4: Gerar e Assinar o APK Modificado para o Celular / Emulador
1. Gere os APKs modificados apontando para a Oracle Cloud:
   ```powershell
   python scripts/gerar_apk_modificado.py --url "https://164.152.40.114:9443" --extracted-dir "CAMINHO_DOS_SPLITS_EXTRAIDOS"
   ```
   *(Importante: Se o Android rejeitar o certificado autoassinado no `UnityWebRequest`, você também pode usar `--url "http://164.152.40.114:9080"` pois o `StringLiteral[10175]` foi atualizado para `^https?://.*$`, permitindo HTTP direto na porta `9080` sem barreiras de SSL/TLS pinning!).*
2. Copie o `split_UnityDataAssetPack.apk` para a pasta `build_apk/` junto com `base.patched.apk` e `split_config.arm64_v8a.patched.apk` e assine todos juntos com a mesma chave usando `uber-apk-signer`:
   ```powershell
   java -jar uber-apk-signer.jar -a build_apk/ --allowResign
   ```
3. Instale os 3 splits assinados no dispositivo Android:
   ```powershell
   adb install-multiple build_apk/*aligned-debugSigned.apk
   ```
