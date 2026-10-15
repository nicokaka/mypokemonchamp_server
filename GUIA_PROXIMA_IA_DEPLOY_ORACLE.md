# 🤖 Guia Técnico & Handoff de Engenharia Reversa — Pokémon Champions (`v1.2.2 Build 3859`)

> **Status Atual (Outubro/2026):**
> - **Servidor Oracle Cloud (`164.152.40.114`):** Ativo e rodando em produção via `pokechamp.service` (`/opt/pokechamp-server/bin/pokechamp-server`), compartilhando a porta pública `443` com o servidor do Pokémon TCG Pocket (`ptcgp.service` em `127.0.0.1:8443`) através do **Smart TCP/TLS/HTTP Multiplexer** na porta `7443`.
> - **MyPokeChamp Studio (`/admin`):** Painel web completo disponível para edição de Perfil, Victory Points (VP), Box Pokémon (IVs/EVs/Moves/Nature/Ability/Tera/Shiny), Times e Bots.
> - **APK Modificado (`build_apk/*-aligned-debugSigned.apk`):** Com bypass de licença Google Play (`LicenseClient`), liberação de HTTP cleartext no `libunity.so` e redirecionamento para `http://164.152.40.114:443`.

---

## 1. Arquitetura na Oracle Cloud (`164.152.40.114`) e Coexistência com PTCGP

A instância ARM64 (`aarch64`) da Oracle Cloud (`ubuntu@164.152.40.114`, chave SSH `/home/nicokaka/Documents/projects/tcg-makemoney/ssh-key-2026-09-19.key`) hospeda dois servidores privados simultaneamente na porta `443`:

1. **Regra `iptables` (`PREROUTING`):**
   - Todo tráfego TCP externo chegando na porta `443` é redirecionado para a porta `7443` (`pokechamp-server`).
2. **Smart Multiplexer (`cmd/server/main.go` na porta `:7443`):**
   - Inspeciona os primeiros bytes de cada conexão TCP (`peekConn`):
     - **HTTP em texto puro (`GET`, `POST`, `HEAD`, `PUT`, `DELETE`, `OPTIONS`):** Atendido diretamente pelo `pokechamp-server` (usado pelo APK modificado do Pokémon Champions em `http://164.152.40.114:443` e pelo painel `/admin`).
     - **TLS `ClientHello` (`0x16`):** Extrai o **SNI** sem consumir o stream:
       - Se o SNI pertencer ao **Pokémon Champions** (`*pokemonchampions*`), termina o TLS localmente com certificado dinâmico (`internal/localtls`).
       - Caso contrário (tráfego do **Pokémon TCG Pocket** ou SNI de `ptcgp`), faz proxy transparente TCP bruto para `127.0.0.1:8443` (`ptcgp.service`).
3. **Comandos Rápidos de Manutenção na Oracle Cloud:**
   ```bash
   # Ver logs em tempo real do Pokémon Champions
   ssh -i /home/nicokaka/Documents/projects/tcg-makemoney/ssh-key-2026-09-19.key ubuntu@164.152.40.114 "sudo journalctl -u pokechamp -f"

   # Compilar localmente para ARM64 e atualizar o binário na Oracle Cloud
   CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o pokechamp-server-arm64 ./cmd/server
   scp -i /home/nicokaka/Documents/projects/tcg-makemoney/ssh-key-2026-09-19.key pokechamp-server-arm64 ubuntu@164.152.40.114:/tmp/pokechamp-server.new
   ssh -i /home/nicokaka/Documents/projects/tcg-makemoney/ssh-key-2026-09-19.key ubuntu@164.152.40.114 "sudo systemctl stop pokechamp && sudo mv /tmp/pokechamp-server.new /opt/pokechamp-server/bin/pokechamp-server && sudo chmod +x /opt/pokechamp-server/bin/pokechamp-server && sudo systemctl start pokechamp"
   ```

---

## 2. Achados Críticos da Engenharia Reversa (`libil2cpp.so` & `global-metadata.dat`)

### 2.1. Por que `resp.Pmc` DEVE SEMPRE ser populado (`Net.ApiTemplate::ParseResponse` — `VA 0x2dd2b58`)
A descompilação de `Net.ApiTemplate<TReq, TReqPmp, TRes, TResPmp>::ParseResponse` (`0x2dd2b58..0x2dd2e58` em `libil2cpp.so`) revelou como a Unity desserializa respostas:
1. Em `0x2dd2c24`, o cliente chama `JsonUtility.FromJson<TRes>(responseText)`. Como `pmp` no wrapper genérico `ResponseBodyBase<TResPmp>` é acessado via propriedades virtuais (`vtable #0x178 get_pmp` / `vtable #0x188 set_pmp`), o `JsonUtility` da Unity **ignora o campo `"pmp"` do JSON externo** e desserializa apenas os campos concretos da classe base `ResponseBodyBase` (como `code`, `pmc`, `tmV`, `ppV`, `wvV`, `sexp`, `inH`).
2. Em `0x2dd2c30..0x2dd2c3c`, o código verifica `string.IsNullOrEmpty(body.pmc)`.
3. **Armadilha:** Se `body.pmc` estiver vazio (`""`), o código salta para `0x2dd2d68` e retorna `(0, body)` **sem nunca chamar `body.set_pmp(...)`**, deixando `body.get_pmp() == null`!
4. Logo em seguida, em `AuthGetToken::CheckResponseBody` (`VA 0x4525950`, instrução `0x45259cc`), o cliente chama `body.get_pmp()`, vê `null` (`cbz x0, 0x4525a2c`) e retorna o código de erro `0x493ed` (`299997`), fazendo o jogo repetir `/auth/get-token` em loop infinito na tela de título!
5. **Solução implementada em [`internal/api/router.go`](internal/api/router.go):**
   - `writeGameResponse` **sempre** criptografa o `pmp` em `resp.Pmc` (e envia `X-PKB-ENCRYPT: 1`), usando `crypto.EncryptPmpAuth` para `/auth/*` e `crypto.EncryptPmpApi` para `/api/*`.
   - Quando `body.pmc` está presente, `ParseResponse` (`0x2dd2ce0..0x2dd2e54`) descriptografa `body.pmc`, converte `:[]` em `:null`, executa `JsonUtility.FromJson<TResPmp>(pmpJson)` e chama `body.set_pmp(pmp)` (`0x2dd2e4c`).

### 2.2. Sincronia de Tempo e `dmy` em `DecryptResponsePmpAuth` (`VA 0x4538b48`)
1. Em `CheckResponseAsync` (`VA 0x45200dc`), o cliente lê primeiro os headers HTTP da resposta:
   - `X-PKB-DMY-VAL` (`0x45202d8`) $\rightarrow$ passado como `dmyVal` para `ParseResponse`.
   - `X-PKB-S-TIME` (`0x452035c`) $\rightarrow$ chama `SetServerTime(sTime)` (`VA 0x454deec`) **antes** de chamar `ParseResponse` (`0x45203fc`).
2. Assim, quando `DecryptResponsePmpAuth(body.pmc, dmyVal)` (`VA 0x4538b48`) roda no cliente, `GetServerTime()` já contém exatamente o `nowUnix` enviado pelo servidor em `X-PKB-S-TIME`!
3. Na recepção de requisições do cliente (`decodeRequestPmp`), `AuthGetToken::CreateRequestBody` (`VA 0x4525628`) gera uma string alfanumérica de 42 caracteres no campo `dmy` do JSON (`env.Dmy`) e um timestamp em milissegundos no header `X-PKB-DMY-VAL`. Nossa função `crypto.DecryptPmcAuth` em [`internal/crypto/crypto.go`](internal/crypto/crypto.go) testa ambos (`dmy` do header e `env.Dmy` do JSON) e também calcula a janela `win = uint32((ms/1000)/600)` a partir do próprio `X-PKB-DMY-VAL` do cliente.

### 2.3. Sincronia de `UdVer = 1` em `DecryptResponsePmpApi` (`VA 0x4539900`)
1. Em `DecryptResponsePmpApi` (`VA 0x4539900`), o cliente tenta descriptografar `body.pmc` das rotas `/api/*` iterando sobre `offsets = {1, 0}` (`0x45399b4`), ou seja, testando `candidateUdVer = clientUdVer + 1` e `clientUdVer + 0`.
2. Antes de carregar o `UD` pela primeira vez, `clientUdVer == 0`. Depois que o `UD` é carregado, `clientUdVer == UD.udVer`.
3. Mantendo `UdVer = 1` fixo em `BuildFullUD()` ([`internal/store/store.go`](internal/store/store.go)) e em `EncryptPmpApi(..., 1)` ([`internal/api/router.go`](internal/api/router.go)), o valor `1` bate **tanto** quando `clientUdVer == 0` (`0 + 1 = 1`) **quanto** quando `clientUdVer == 1` (`1 + 0 = 1`), eliminando qualquer risco de dessincronização mesmo após edições no painel `/admin`.

### 2.4. Exigência de `UUID` com Exatamente 42 Caracteres (`AuthRegisterUser::CheckResponseBody` — `VA 0x4525b3c`)
1. Em `AuthRegisterUser::CheckResponseBody` (`0x4525c10..0x4525c18`), o cliente executa `cmp w8, #0x2a` (`42`) sobre `pmp.uuid.Length`.
2. Se `len(uuid) != 42`, o cliente rejeita a resposta com erro `0x493eb` (`299995`).
3. Em [`internal/store/store.go`](internal/store/store.go), `normalizeUUID42` garante que `s.state.UUID` e `s.state.Profile.Id` tenham sempre 42 caracteres (`"champ0001nicolas0001pokemonchampions000001"` por padrão, ou o UUID de 42 caracteres enviado pelo próprio cliente).

### 2.5. Arquitetura de Master Data (`/md/c/...`) e AssetBundles Offline (`split_UnityDataAssetPack.apk`)
1. **100% dos 2.647 AssetBundles (`1,14 GB`) são Built-in no APK:**
   - O catálogo `assets/AssetAssistant/syn.bin` dentro de `split_UnityDataAssetPack.apk` contém 2.603 registros (`_records`), todos marcados com `"isBuiltin": true` (`0` pacotes remotos).
   - O passo *"Downloading game data"* na tela de título baixa exclusivamente as **89 tabelas JSON criptografadas de Master Data (`Net.Md*Param`)** servidas por [`internal/api/md.go`](internal/api/md.go) em `GET /md/c/{sha1_hash}`.
2. **Proibição de Tabelas Vazias (`"[]"`) por causa de `ReadMDParam::GetTargetData<T>` (`VA 0x2cd0368`):**
   - Quase todos os getters de `ReadMDParam` (`0x23c8418..0x23d1adc`) delegam para `GetTargetData<T>(T[] list, Predicate<T> match)` (`0x2cd0368`), que faz `return list[0];` (`0x2cd03f0`) caso o ID buscado não seja encontrado, e lança `IndexOutOfRangeException` (`0x2cd0414`) se `list.Length == 0` (`"[]"`).
   - O gerador [`scripts/generate_master_data.py`](scripts/generate_master_data.py) extrai diretamente dos AssetBundles do APK (`uisprites/personal/s`, `uisprites/avatarcostume/males`, `trainer/avatar/partsdata`, `uisprites/item/s`, etc.) todas as **89 tabelas JSON (`5.033` registros)** em `game-data/master-data/*.json`, garantindo que **nenhuma** tabela seja `"[]"`.
3. **Validação de Avatar (`PlayerAvatar::ValidateParts` — `VA 0x24c0bd8`) e Tutorial (`TuPrg = 1200`):**
   - Logo após baixar a 89ª tabela (`junior_ignore_resource`), `TitleScene::StartLoadAssetsState` (`VA 0x23b38e8`) chama `PlayerAvatar::Create` (`VA 0x24bf804`) $\rightarrow$ `PlayerAvatar::ValidateParts` (`VA 0x24c0bd8`).
   - Os IDs válidos do Avatar masculino padrão (`id=1` em `AvatarHelper::CreateInitialAvatarParam`, `VA 0x24bda18`) configurados em [`internal/store/store.go`](internal/store/store.go) (`currentSchemaVer = 3`) são:
     - `Cos`: `{"p0": 1, "p0c": 1, "p1": 1001, "p2": 0, "p3": 5001, "p4": 7001, "p5": 9001, "p6": 11001, "p7": 13001, "p8": 15001, "p9": 0, "pa": 0, "pb": 0}`
     - `Fac`: `{"p0": 25001, "p0c": 5, "p1": 23022, "p2": 27001, "p2c": 5, "p3": 29001, "p4": 31003, "p5": 0, "p6": 0, "p7": 37001, "p7c": 32, "p8c": 1, "p8l": 42}`
     - `Mot`: `{"p0": 1001, "p1": 3001, "p2": 5001}`
   - O checkpoint final de tutorial em `EventScriptCheckPoint` (bundle `eventscript`) é **`TuPrg = 1200`** e `SubEventManager::IsAllBattleTutorial` (`VA 0x26289ac`) verifica as flags **`400..413`** em `TuFlgs`. Com ambos preenchidos, `TitleScene::StartValidateIntroductionState` (`VA 0x23b4958`) navega direto para a **`HomeScene`**.

---

## 3. Patches Aplicados no APK (`scripts/gerar_apk_modificado.py`)

O gerador [`scripts/gerar_apk_modificado.py`](scripts/gerar_apk_modificado.py) aplica 4 camadas de patch para produzir os splits em `build_apk/`:

1. **Bypass da Verificação de Licença da Google Play (`base.apk` $\rightarrow$ `classes.dex`):**
   - O APK original verifica a assinatura/licença da Google Play em `com.google.android.vending.licensing.LicenseClient::checkLicense`. Quando re-assinado com chave debug, o método original trava o jogo em uma tela preta logo na abertura.
   - O script substitui o bytecode Dalvik de `checkLicense` por `return-void` (`0e 00`) e recalcula os headers SHA-1 e Adler32 do `classes.dex`.
2. **Reconstrução da Tabela de Strings do `global-metadata.dat` (`base.apk`):**
   - `StringLiteral[13153]`: `"https://api.app.pokemonchampions.jp"` $\rightarrow$ `"http://164.152.40.114:443"`
   - `StringLiteral[10175]`: Regex de validação de `envLs` $\rightarrow$ `"^https?://.*$"`
   - `StringLiteral[10176]`: Segundo Regex de `SetApiEndpoint` $\rightarrow$ `"^https?://.*$"`
3. **Desbloqueio de HTTP Cleartext no `libunity.so` (`split_config.arm64_v8a.apk`):**
   - O `UnityWebRequest` nativo bloqueia URLs iniciadas por `"http://"` quando a flag de segurança está ativa. O patcher altera a string literal `"http://"` do validador de bloqueio em `libunity.so` para `"xttp://"`, permitindo tráfego HTTP direto na porta `443` sem exigir certificado Root CA no Android.
4. **Patches em `libil2cpp.so` (`split_config.arm64_v8a.apk`):**
   - Neutraliza checagens restritivas de hash/ambiente em `Net.dll`.

### Como Regerar e Instalar o APK no Celular via ADB
```bash
# 1. Gerar e assinar os 3 splits em build_apk/
python3 scripts/gerar_apk_modificado.py --url "http://164.152.40.114:443"

# 2. Instalar no celular conectado via USB
./instalar_no_celular.sh
```
