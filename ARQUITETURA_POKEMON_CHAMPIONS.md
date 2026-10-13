# Arquitetura Técnica & Manual de Engenharia Reversa: Pokémon Champions

**Projeto:** Pokémon Champions (`jp.pokemon.pokemonchampions`) — Servidor Privado  
**Versão do Jogo:** v1.2.2 (Build 3859)  
**Engine:** Unity 2022.3 / 2023.1 LTS (IL2CPP ARM64)  
**Domínio Oficial:** `https://api.app.pokemonchampions.jp`  

---

## 1. Topologia de Rede e Protocolo

Ao contrário do Pokémon TCG Pocket, o Pokémon Champions utiliza **JSON estruturado sobre HTTP/HTTPS** em conjunto com a biblioteca `NetUtf8Json.dll` e `Utf8Json.dll`.

### Formato de Requisição & Resposta
* Todas as rotas de API possuem contratos DTOs tipados no namespace `Net`:
  * `Net.Api{Nome}RequestParam`
  * `Net.Api{Nome}ResponseParam`
* O cliente serializa o DTO em JSON, compacta via GZip (`GZipEncode`) e cifra via AES (`AesEncrypt`).

---

## 2. Cabeçalhos de Segurança Customizados (Headers `X-PKB-*`)

Identificados na classe `EncryptUtils` do cliente:

| Cabeçalho | Formato / Padrão | Finalidade |
| :--- | :--- | :--- |
| `X-PKB-ENCRYPT` | `[{0}{1}{2}]` | Parâmetros de identificação da cifra e tokens de sessão |
| `X-PKB-HASH-CHECK` | String hexadecimal | Hash HMAC/SHA de integridade da carga |
| `X-PKB-S-TIME` | `[{0}]` ou timestamp | Carimbo de data/hora sincronizado com o servidor |
| `X-PKB-DMY-VAL` | String aleatória | Valor dummy anti-replay |

---

## 3. Catálogo Canônico de Endpoints da API

### 3.1. Autenticação (`/auth/`)
* `/auth/register-user`: Criação de novo usuário / perfil
* `/auth/login`: Autenticação e validação de sessão
* `/auth/get-token`: Renovação de token JWT/sessão
* `/auth/check-register-limit`: Checagem de limites de cadastro
* `/auth/competition-mode-login`: Login em modo de torneio
* `/auth/takeover/get-na-link-params`: Integração com Nintendo Account (`NA`)
* `/auth/takeover/get-apple-link-params`: Integração com Apple ID

### 3.2. Sincronização de Estado (`/api/common/`)
* `/api/common/get-all-ud`: Retorna todo o estado do usuário (`UD = User Data`)
* `/api/common/regular-check`: Heartbeat periódico e checagem de integridade
* `/api/user/update`: Atualização de informações de perfil

### 3.3. Armazenamento de Pokémon (`/api/box/`)
* `/api/box/nickname`: Alteração de apelido de Pokémon
* `/api/box/ribbon`: Equipar/gerenciar fitas (Ribbons)
* `/api/box/goodbye` & `/api/box/goodbye-check`: Liberação / despedida de Pokémon
* `/api/box/label/create`, `/update`, `/delete`, `/set`: Gerenciamento de etiquetas

### 3.4. Equipes & Treinamento (`/api/team/` e `/api/training/`)
* `/api/team/update`: Salvar time competitivo
* `/api/team/delete`: Excluir time competitivo
* `/api/training/update`: Atualizar atributos e treinamento de Pokémon
* `/api/trainingcode/register`, `/search`, `/update`, `/delete`: Códigos de equipe compartilhados

### 3.5. Batalhas, Matchmaking & Lobbies (`/api/battle/` e `/api/lobby/`)
* `/api/battle/start-matching`: Entrar na fila de pareamento
* `/api/battle/resume-matching`: Retomar busca de partida
* `/api/battle/start-battle`: Início da partida
* `/api/battle/end-battle`: Conclusão e registro do placar
* `/api/battle/surrender-battle`: Rendição / desistência
* `/api/lobby/create`, `/join`, `/leave`, `/search`: Lobbies de partida privada

### 3.6. Torneios & Eventos (`/api/competition/` e `/api/ranking/`)
* `/api/competition/list`: Lista de campeonatos abertos
* `/api/competition/entry` & `/cancel-entry`: Inscrição em torneio
* `/api/competition/confirm-result`: Validação de resultado de torneio
* `/api/ranking/competition/get-ranking`: Leaderboard do torneio
* `/api/ranking/regular-event/get-ranking`: Leaderboard de eventos regulares

---

## 4. Estratégia de Hospedagem Gratuita (Oracle Cloud Always Free)

Consulte a seção técnica para configurar o servidor na mesma VM da Oracle (com porta e serviço isolados) ou em uma nova instância ARM, sem nenhum custo financeiro.
