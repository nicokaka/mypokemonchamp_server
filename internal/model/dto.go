package model

import "encoding/json"

// ApiRequestEnvelope represents Net.ApiRequestBodyBase + pmp (Type 13519).
type ApiRequestEnvelope struct {
	ClV  string          `json:"clV,omitempty"`
	MdV  *int            `json:"mdV,omitempty"`
	Pla  int             `json:"pla,omitempty"`
	TmV  uint32          `json:"tmV,omitempty"`
	PpV  uint32          `json:"ppV,omitempty"`
	WvV  uint32          `json:"wvV,omitempty"`
	InH  string          `json:"inH,omitempty"`
	Lng  int             `json:"lng,omitempty"`
	Cou  int             `json:"cou,omitempty"`
	Jrom int             `json:"jrom,omitempty"`
	Hsh  string          `json:"hsh,omitempty"`
	Sid  string          `json:"sid,omitempty"`
	Pmc  string          `json:"pmc,omitempty"`
	AsNV string          `json:"asNV,omitempty"`
	Pmp  json.RawMessage `json:"pmp,omitempty"`
}

// ApiResponseEnvelope represents Net.ApiResponseBodyBase + pmp (Type 13520).
type ApiResponseEnvelope struct {
	Code    int                  `json:"code"`
	Esecode string               `json:"esecode,omitempty"`
	Esetype string               `json:"esetype,omitempty"`
	Pmc     string               `json:"pmc,omitempty"`
	Msg     string               `json:"msg,omitempty"`
	Md      *MdMeta              `json:"md,omitempty"`
	TmV     *uint32              `json:"tmV,omitempty"`
	TmU     string               `json:"tmU,omitempty"`
	PpV     *uint32              `json:"ppV,omitempty"`
	PpU     string               `json:"ppU,omitempty"`
	WvV     *uint32              `json:"wvV,omitempty"`
	WvU     string               `json:"wvU,omitempty"`
	Sid     string               `json:"sid,omitempty"`
	Sexp    *int                 `json:"sexp,omitempty"`
	RqUrl   string               `json:"rqUrl,omitempty"`
	InH     string               `json:"inH,omitempty"`
	InU     string               `json:"inU,omitempty"`
	NclV    string               `json:"nclV,omitempty"`
	MaV     *int                 `json:"maV,omitempty"`
	Cocl    *int                 `json:"cocl,omitempty"`
	Asset   *ResponseAssetBundle `json:"asset,omitempty"`
	MMode   *int                 `json:"mMode,omitempty"`
	MSt     *uint64              `json:"mSt,omitempty"`
	MEd     *uint64              `json:"mEd,omitempty"`
	Mlng    map[string]string    `json:"mlng,omitempty"`
	Frmsg   map[string]string    `json:"frmsg,omitempty"`
	Frurls  []string             `json:"frurls,omitempty"`
	Pmp     interface{}          `json:"pmp,omitempty"`
}

type MdMeta struct {
	Ver int               `json:"ver"`
	Dom string            `json:"dom"`
	Fls map[string]string `json:"fls,omitempty"`
}

type ResponseAssetBundle struct {
	Ver string `json:"ver"`
	Dom string `json:"dom"`
}

// AuthEnv represents Net.AuthGetTokenResponseParam.Env (Type 14037).
type AuthEnv struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

// AuthGetTokenPmp represents Net.AuthGetTokenResponseParam.UniqueParam (Type 14036).
type AuthGetTokenPmp struct {
	Token string    `json:"token"`
	Exp   uint64    `json:"exp"`
	EnvLs []AuthEnv `json:"envLs"`
}

// AuthRegisterUserPmp represents Net.ApiAuthRegisterUserResponseParam.UniqueParam (Type 14047).
type AuthRegisterUserPmp struct {
	Uuid   string        `json:"uuid"`
	Supid  string        `json:"supid"`
	Dvcode string        `json:"dvcode"`
	Lista  []interface{} `json:"lista"`
	UD     *ApiUdParam   `json:"UD,omitempty"`
}

type LoginTimeOffset struct {
	Offset int  `json:"offset"`
	Isfix  byte `json:"isfix"`
}

// AuthLoginPmp represents Net.ApiLoginResponseParam.UniqueParam (Type 14041).
type AuthLoginPmp struct {
	Sid      string          `json:"sid"`
	Sexp     int             `json:"sexp"`
	Refnotif []interface{}   `json:"refnotif"`
	Tmoff    LoginTimeOffset `json:"tmoff"`
	Phtrbx   *int            `json:"phtrbx,omitempty"`
	Phtrul   *int            `json:"phtrul,omitempty"`
	Btl      byte            `json:"btl"`
	Lby      byte            `json:"lby"`
	UD       *ApiUdParam     `json:"UD,omitempty"`
}

type RegularCheckNotification struct {
	Wldq byte `json:"wldq"`
	Eve  byte `json:"eve"`
}

// CommonRegularCheckPmp represents Net.ApiCommonRegularCheckResponseParam.UniqueParam (Type 13661).
type CommonRegularCheckPmp struct {
	Exmsg  byte                     `json:"exmsg"`
	Exmsgc int                      `json:"exmsgc"`
	Frireq int                      `json:"frireq"`
	Eveids []string                 `json:"eveids"`
	Comids []string                 `json:"comids"`
	Comntf RegularCheckNotification `json:"comntf"`
	Phtrbx int                      `json:"phtrbx"`
	Ban    map[string]interface{}   `json:"ban"`
	Revd   int                      `json:"revd"`
	UD     *ApiUdParam              `json:"UD,omitempty"`
}

// UdUpdate wraps single-object UD sections (Net.ApiUdUpdateParam<T>, Type 14081).
type UdUpdate[T any] struct {
	All *T `json:"all,omitempty"`
	Upd *T `json:"upd,omitempty"`
}

// UdDiffMap wraps dictionary UD sections (Net.ApiDiffParams<K,V>, Type 14086).
type UdDiffMap[T any] struct {
	All map[string]T `json:"all,omitempty"`
	Upd map[string]T `json:"upd,omitempty"`
	Del []string     `json:"del,omitempty"`
}

// ApiUdParam represents Net.ApiUdParam (Type 13530).
type ApiUdParam struct {
	Prf      *UdUpdate[ApiUserProfileParam]       `json:"prf,omitempty"`
	Main     *UdUpdate[ApiUserMainParam]          `json:"main,omitempty"`
	Team     *UdDiffMap[ApiUserTeamParam]         `json:"team,omitempty"`
	Avat     *UdUpdate[ApiUserAvatarParam]        `json:"avat,omitempty"`
	Pokes    *UdDiffMap[ApiUserPokemonParam]      `json:"pokes,omitempty"`
	Label    map[string]interface{}               `json:"label,omitempty"`
	Ssndt    map[string]interface{}               `json:"ssndt,omitempty"`
	Btrk     map[string]interface{}               `json:"btrk,omitempty"`
	Club     map[string]interface{}               `json:"club,omitempty"`
	Tng      *UdDiffMap[ApiUserTrainingCodeParam] `json:"tng,omitempty"`
	Avatp    map[string]interface{}               `json:"avatp,omitempty"`
	Invent   map[string]interface{}               `json:"invent,omitempty"`
	Miss     map[string]interface{}               `json:"miss,omitempty"`
	Tachv    map[string]interface{}               `json:"tachv,omitempty"`
	Pachv    map[string]interface{}               `json:"pachv,omitempty"`
	Shopdt   map[string]interface{}               `json:"shopdt,omitempty"`
	Prfsumdt map[string]interface{}               `json:"prfsumdt,omitempty"`
	Usedhis  map[string]interface{}               `json:"usedhis,omitempty"`
	Freebox  map[string]interface{}               `json:"freebox,omitempty"`
	UdVer    int                                  `json:"udVer"`
}

// ApiUserProfileParam represents Net.ApiUserProfileParam (Type 14141).
type ApiUserProfileParam struct {
	AgTA   uint64 `json:"agTA"`
	Bth    int    `json:"bth"`
	Clogc  int    `json:"clogc"`
	Cou    int    `json:"cou"`
	CreAt  uint64 `json:"creAt"`
	Flng   int    `json:"flng"`
	Fri    string `json:"fri"`
	Id     string `json:"id"`
	Lavsp  int    `json:"lavsp"`
	Lavsw  int    `json:"lavsw"`
	Llday  uint64 `json:"llday"`
	Lldaz  uint64 `json:"lldaz"`
	Llog   uint64 `json:"llog"`
	Lng    int    `json:"lng"`
	Mode   int    `json:"mode"`
	Motim  uint64 `json:"motim"`
	Nam    string `json:"nam"`
	PpV    uint32 `json:"ppV"`
	Push   int    `json:"push"`
	Sup    string `json:"sup"`
	Tlogc  int    `json:"tlogc"`
	TmV    uint32 `json:"tmV"`
	Tzof   int    `json:"tzof"`
	Delsts int    `json:"delsts"`
	DeTok  string `json:"deTok"`
	Dv     string `json:"dv"`
	Os     string `json:"os"`
	Trtit  int    `json:"trtit"`
	Trico  int    `json:"trico"`
	Trbad  int    `json:"trbad"`
	TrnId  string `json:"trnId"`
	TrCAt  uint64 `json:"trCAt"`
	TrIAt  uint64 `json:"trIAt"`
	UpRAt  uint64 `json:"upRAt"`
	UpRCt  int    `json:"upRCt"`
	SwPNam string `json:"swPNam"`
	SwBtDt uint64 `json:"swBtDt"`
	SwPrDt int    `json:"swPrDt"`
	AcFrRq int    `json:"acFrRq"`
	UplAt  uint64 `json:"uplAt"`
	HbuNam string `json:"hbuNam"`
}

// ApiUserMainParam represents Net.ApiUserMainParam (Type 14123).
type ApiUserMainParam struct {
	Vp       int                    `json:"vp"`
	TuPrg    int                    `json:"tuPrg"`
	TuFlgs   []int                  `json:"tuFlgs"`
	Pbsiz    int                    `json:"pbsiz"`
	Pbexc    int                    `json:"pbexc"`
	Pcnt     int                    `json:"pcnt"`
	Ssnno    int                    `json:"ssnno"`
	Bpasi    map[string]interface{} `json:"bpasi"`
	Subsc    map[string]interface{} `json:"subsc"`
	Misri    map[string]interface{} `json:"misri"`
	Emisi    map[string]interface{} `json:"emisi"`
	Putoi    []interface{}          `json:"putoi"`
	Ptids    []int                  `json:"ptids"`
	Nrmsgid  string                 `json:"nrmsgid"`
	Lrlcst   uint64                 `json:"lrlcst"`
	Mxrnk    int                    `json:"mxrnk"`
	Nwnver   int                    `json:"nwnver"`
	Ngunmflg int                    `json:"ngunmflg"`
	Vpbcnt   int                    `json:"vpbcnt"`
	Fcbs     int                    `json:"fcbs"`
	Fpbs     int                    `json:"fpbs"`
	Swpai    map[string]interface{} `json:"swpai"`
}

type ApiUserAvatarParam struct {
	Fac map[string]int `json:"fac"`
	Cos map[string]int `json:"cos"`
	Mot map[string]int `json:"mot"`
}

type ApiUserTeamItemParam struct {
	P string `json:"p"`
	I int    `json:"i"`
}

// ApiUserTeamParam represents Net.ApiUserTeamParam (Type 14168).
type ApiUserTeamParam struct {
	Tid    string                 `json:"tid"`
	Nam    string                 `json:"nam"`
	Lng    int                    `json:"lng"`
	Mem    []string               `json:"mem"`
	Cuflg  int                    `json:"cuflg"`
	Dcuflg int                    `json:"dcuflg"`
	Itms   []ApiUserTeamItemParam `json:"itms"`
}

// ApiUserPokemonParam represents Net.ApiUserPokemonParam (Type 14138).
type ApiUserPokemonParam struct {
	Ver int      `json:"ver"`
	A0  string   `json:"a0"` // Origin ID / Trainer ID
	A1  string   `json:"a1"` // Unique ID
	A5  byte     `json:"a5"` // Origin Type / Rental flag
	A6  uint64   `json:"a6"` // CreatedAt
	A7  uint64   `json:"a7"` // UpdatedAt
	A8  uint64   `json:"a8"` // Rental Expire
	A9  string   `json:"a9"` // HOME ID
	Aa  uint64   `json:"aa"` // Timestamp
	B0  int      `json:"b0"` // Encryption Constant / PID
	B1  uint16   `json:"b1"` // National Dex ID (MonsNo)
	B2  byte     `json:"b2"` // Form No
	B3  byte     `json:"b3"` // Gender (0=M, 1=F, 2=Genderless)
	B4  int      `json:"b4"` // Experience / Level50
	B5  uint16   `json:"b5"` // Ability ID (Tokusei)
	B6  []uint16 `json:"b6"` // Active Moves (Waza 1..4)
	B7  uint16   `json:"b7"` // Held Item
	B8  uint16   `json:"b8"` // PokeBall ID
	B9  byte     `json:"b9"` // EV HP (BasePoint)
	Ba  byte     `json:"ba"` // EV Atk
	Bb  byte     `json:"bb"` // EV Def
	Bc  byte     `json:"bc"` // EV Spd
	Bd  byte     `json:"bd"` // EV SpAtk
	Be  byte     `json:"be"` // EV SpDef
	Bf  []byte   `json:"bf"` // IVs (6 bytes: 31,31,31,31,31,31)
	Bg  []int    `json:"bg"` // Move PP Ups
	Bh  []uint16 `json:"bh"` // Learned Moves
	Bi  uint16   `json:"bi"` // Initial Ability
	Bj  byte     `json:"bj"` // Nature (Seikaku)
	Bk  byte     `json:"bk"` // Mint Nature (Seikaku Hosei)
	Bl  byte     `json:"bl"` // Shiny Flag (0=Normal, 1=Shiny)
	Bm  int      `json:"bm"` // Tera Type / Gimmick Type
	Bn  byte     `json:"bn"` // Language
	Bo  string   `json:"bo"` // Nickname
	Bp  byte     `json:"bp"` // Nickname Flag
	Bt  uint16   `json:"bt"` // Height / Scale
	Bu  uint16   `json:"bu"` // Weight
	Bv  []int    `json:"bv"` // Ribbons
	Bw  byte     `json:"bw"` //Equipped Ribbon
	Bx  string   `json:"bx"` // Original Trainer Name (OT)
	By  byte     `json:"by"` // OT Gender
	I   int      `json:"i"`  // Box Slot Index
}

type ApiUserTrainingCodeParam struct {
	Ver   int                    `json:"ver"`
	Tnam  string                 `json:"tnam"`
	Tlng  int                    `json:"tlng"`
	Unam  string                 `json:"unam"`
	Ulng  int                    `json:"ulng"`
	Trico int                    `json:"trico"`
	Mem   []string               `json:"mem"`
	Itms  []ApiUserTeamItemParam `json:"itms"`
	Fav   byte                   `json:"fav"`
	CreAt uint64                 `json:"creAt"`
}

// BattleStartMatchingPmp represents Net.ApiBattleStartMatchingResponseParam.UniqueParam (Type 13616).
type BattleStartMatchingPmp struct {
	Ws   string      `json:"ws"`
	Atk  string      `json:"atk"`
	Exp  uint64      `json:"exp"`
	Ehsh string      `json:"ehsh"`
	UD   *ApiUdParam `json:"UD,omitempty"`
}

// BattleStartBattlePmp represents Net.ApiBattleStartBattleResponseParam.UniqueParam (Type 13611).
type BattleStartBattlePmp struct {
	Rmid   string      `json:"rmid"`
	Sprule int         `json:"sprule"`
	UD     *ApiUdParam `json:"UD,omitempty"`
}

// BattleEndBattlePmp represents Net.ApiBattleEndBattleResponseParam.UniqueParam (Type 13583).
type BattleEndBattlePmp struct {
	Bca int           `json:"bca"`
	Rul int           `json:"rul"`
	Rw  []interface{} `json:"rw"`
	UD  *ApiUdParam   `json:"UD,omitempty"`
}

// MdBotMergeParam represents Net.MdBotMergeParam (Type 14435) for AI opponents.
type MdBotMergeParam struct {
	ID          int      `json:"id"`
	TrainerName string   `json:"trainer_name"`
	Use         int      `json:"use"`
	Lose        int      `json:"lose"`
	Ra          int      `json:"ra"`
	Ira         int      `json:"ira"`
	Avatar      int      `json:"avatar"`
	Mons        []uint16 `json:"mons"`
}
