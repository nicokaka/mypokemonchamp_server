# MyPokeChampServer - Pokémon Champions Private Server & Toolchain

Servidor privado, emulador de backend e ferramental de engenharia reversa para o jogo **Pokémon Champions (`jp.pokemon.pokemonchampions`)**.

---

## 🎯 Visão Geral do Projeto

O **MyPokeChampServer** emula a infraestrutura de backend oficial (`https://api.app.pokemonchampions.jp`), permitindo:
* Autenticação e criação de perfil offline/privado;
* Sincronização completa de dados do jogador (`User Data / UD`);
* Gerenciamento de caixas de Pokémon (Box Storage), times e códigos de treino;
* Partidas, matchmaking de arena e lobbies;
* Emulação de loja de itens e passes de batalha.

---

## 🏗️ Arquitetura Técnica

Ao contrário do Pokémon TCG Pocket (que utiliza gRPC/Protobuf e contêineres CySharp MasterMemory proprietários), o **Pokémon Champions** adota uma arquitetura baseada em:

1. **Protocolo:** REST sobre HTTP/HTTPS com serialização JSON estruturada (`NetUtf8Json` / `Utf8Json`).
2. **Camada de Criptografia (`EncryptUtils`):**
   - Criptografia simétrica AES-CBC dos payloads de requisição e resposta;
   - Compressão GZip automática (`GZipEncode` / `GZipDecode`);
   - Cabeçalhos de segurança customizados:
     - `X-PKB-ENCRYPT`: `[{0}{1}{2}]`
     - `X-PKB-HASH-CHECK`: Assinatura / Verificação de integridade
     - `X-PKB-S-TIME`: Timestamp de sincronismo de sessão
     - `X-PKB-DMY-VAL`: Valor dummy de proteção contra replay
3. **Ambiente de Rede (`NetEnvironment`):**
   - Endpoints configuráveis no cliente Unity através de `SetApiEndpoint` / `DefaultApiEndPoint`.
4. **Backend Privado:** Desenvolvido em **Go (Golang)** de altíssima performance e baixo consumo de memória.

---

## 📁 Estrutura do Repositório

```text
mypokechamp_server/
├── cmd/
│   └── server/             # Ponto de entrada do executável principal
├── internal/
│   ├── api/                # Handlers REST (/api/common, /api/box, /api/battle, etc.)
│   ├── auth/               # Handlers de autenticação (/auth/login, /auth/get-token, etc.)
│   ├── crypto/             # AES-CBC, GZip e gerador de cabeçalhos X-PKB
│   ├── model/              # DTOs de Request e Response (Net.Api*Param)
│   └── store/              # Persistência de dados (SQLite em modo WAL)
├── docs/                   # Especificações e engenharia reversa
├── ARQUITETURA_POKEMON_CHAMPIONS.md # Manual técnico detalhado
├── go.mod
└── README.md
```

---

## 🚀 Como Executar

### Pré-requisitos
* Go 1.22 ou superior

### Inicialização
```bash
go run ./cmd/server
```
