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
	Dmy  string          `json:"dmy,omitempty"`
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
	Uuid   string          `json:"uuid"`
	Supid  string          `json:"supid"`
	Dvcode string          `json:"dvcode"`
	Lista  map[string]byte `json:"lista"`
	UD     *ApiUdParam     `json:"UD,omitempty"`
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
	UdVer    int64                                `json:"udVer"`
}

// ApiUserProfileParam represents Net.ApiUserProfileParam (Type 14141).
type ApiUserProfileParam struct {
	AgTA   int64    `json:"agTA"`
	Bth    int      `json:"bth"`
	Clogc  int64    `json:"clogc"`
	Cou    int      `json:"cou"`
	CreAt  int64    `json:"creAt"`
	Flng   int      `json:"flng"`
	Fri    string   `json:"fri"`
	Id     string   `json:"id"`
	Lavsp  string   `json:"lavsp"`
	Lavsw  string   `json:"lavsw"`
	Llday  int      `json:"llday"`
	Lldaz  int      `json:"lldaz"`
	Llog   int64    `json:"llog"`
	Lng    int      `json:"lng"`
	Mode   int      `json:"mode"`
	Motim  int64    `json:"motim"`
	Nam    string   `json:"nam"`
	PpV    int      `json:"ppV"`
	Push   byte     `json:"push"`
	Sup    string   `json:"sup"`
	Tlogc  int64    `json:"tlogc"`
	TmV    int      `json:"tmV"`
	Tzof   int      `json:"tzof"`
	Delsts int      `json:"delsts"`
	DeTok  string   `json:"deTok"`
	Dv     string   `json:"dv"`
	Os     string   `json:"os"`
	Trtit  string   `json:"trtit"`
	Trico  string   `json:"trico"`
	Trbad  []string `json:"trbad"`
	TrnId  string   `json:"trnId"`
	TrCAt  uint64   `json:"trCAt"`
	TrIAt  uint64   `json:"trIAt"`
	UpRAt  uint64   `json:"upRAt"`
	UpRCt  int      `json:"upRCt"`
	SwPNam byte     `json:"swPNam"`
	SwBtDt byte     `json:"swBtDt"`
	SwPrDt byte     `json:"swPrDt"`
	AcFrRq byte     `json:"acFrRq"`
	UplAt  uint64   `json:"uplAt"`
	HbuNam byte     `json:"hbuNam"`
}

// ApiUserMainParam represents Net.ApiUserMainParam (Type 14123).
type ApiUserMainParam struct {
	Vp       int64                         `json:"vp"`
	TuPrg    int                           `json:"tuPrg"`
	TuFlgs   []int                         `json:"tuFlgs"`
	Pbsiz    int                           `json:"pbsiz"`
	Pbexc    int                           `json:"pbexc"`
	Pcnt     int                           `json:"pcnt"`
	Ssnno    int                           `json:"ssnno"`
	Bpasi    map[string]interface{}        `json:"bpasi"`
	Subsc    map[string]interface{}        `json:"subsc"`
	Misri    map[string]interface{}        `json:"misri"`
	Emisi    map[string]interface{}        `json:"emisi"`
	Putoi    map[string]map[string]int     `json:"putoi"`
	Ptids    []interface{}                 `json:"ptids"`
	Nrmsgid  string                        `json:"nrmsgid"`
	Lrlcst   uint64                        `json:"lrlcst"`
	Mxrnk    uint32                        `json:"mxrnk"`
	Nwnver   uint32                        `json:"nwnver"`
	Ngunmflg byte                          `json:"ngunmflg"`
	Vpbcnt   int                           `json:"vpbcnt"`
	Fcbs     int64                         `json:"fcbs"`
	Fpbs     int64                         `json:"fpbs"`
	Swpai    map[string]interface{}        `json:"swpai"`
}

// ApiUserAvatarParam represents Net.ApiUserAvatarParam (Type 14092).
type ApiUserAvatarParam struct {
	Fac map[string]int `json:"fac"`
	Cos map[string]int `json:"cos"`
	Mot map[string]int `json:"mot"`
}

// ApiUserTeamItemParam represents Net.ApiUserTeamItemParam (Type 14164).
type ApiUserTeamItemParam struct {
	P string `json:"p"`
	I int    `json:"i"`
}

// ApiUserTeamParam represents Net.ApiUserTeamParam (Type 14168).
type ApiUserTeamParam struct {
	Tid    int                    `json:"tid"`
	Nam    string                 `json:"nam"`
	Lng    int                    `json:"lng"`
	Mem    []string               `json:"mem"`
	Cuflg  int                    `json:"cuflg"`
	Dcuflg int                    `json:"dcuflg"`
	Itms   []ApiUserTeamItemParam `json:"itms"`
}

// ApiUserPokemonParam represents Net.ApiUserPokemonParam (Type 14138).
// Note: Bf MUST be []uint16 (ushort[]) and NOT []byte, because Go's encoding/json
// encodes []byte as a Base64 string instead of a JSON integer array!
type ApiUserPokemonParam struct {
	Ver uint32   `json:"ver"`
	A0  string   `json:"a0"` // Origin ID / Trainer ID
	A1  string   `json:"a1"` // Unique ID
	A5  byte     `json:"a5"` // Origin Type / Rental flag
	A6  uint64   `json:"a6"` // CreatedAt
	A7  uint64   `json:"a7"` // UpdatedAt
	A8  uint64   `json:"a8"` // Rental Expire
	A9  string   `json:"a9"` // HOME ID
	Aa  uint64   `json:"aa"` // Timestamp
	B0  uint32   `json:"b0"` // Encryption Constant / PID
	B1  uint16   `json:"b1"` // National Dex ID (MonsNo)
	B2  byte     `json:"b2"` // Form No
	B3  byte     `json:"b3"` // Gender (0=M, 1=F, 2=Genderless)
	B4  uint32   `json:"b4"` // Experience / Level50
	B5  uint16   `json:"b5"` // Ability ID (Tokusei)
	B6  []int16  `json:"b6"` // Active Moves (Waza 1..4) - short[]
	B7  uint16   `json:"b7"` // Held Item
	B8  uint16   `json:"b8"` // PokeBall ID
	B9  byte     `json:"b9"` // EV HP (BasePoint)
	Ba  byte     `json:"ba"` // EV Atk
	Bb  byte     `json:"bb"` // EV Def
	Bc  byte     `json:"bc"` // EV Spd
	Bd  byte     `json:"bd"` // EV SpAtk
	Be  byte     `json:"be"` // EV SpDef
	Bf  []uint16 `json:"bf"` // IVs (ushort[6]: 31,31,31,31,31,31)
	Bg  []int64  `json:"bg"` // Move PP Ups - long[]
	Bh  []int16  `json:"bh"` // Learned Moves - short[]
	Bi  uint16   `json:"bi"` // Initial Ability
	Bj  byte     `json:"bj"` // Nature (Seikaku)
	Bk  byte     `json:"bk"` // Mint Nature (Seikaku Hosei)
	Bl  byte     `json:"bl"` // Shiny Flag (0=Normal, 1=Shiny)
	Bm  uint32   `json:"bm"` // Tera Type / Gimmick Type
	Bn  byte     `json:"bn"` // Language
	Bo  string   `json:"bo"` // Nickname
	Bp  byte     `json:"bp"` // Nickname Flag
	Bt  uint16   `json:"bt"` // Height / Scale
	Bu  uint16   `json:"bu"` // Weight
	Bv  []int64  `json:"bv"` // Ribbons - long[]
	Bw  byte     `json:"bw"` // Equipped Ribbon
	Bx  string   `json:"bx"` // Original Trainer Name (OT)
	By  byte     `json:"by"` // OT Gender
	I   uint32   `json:"i"`  // Box Slot Index
}

// ApiUserTrainingCodeItemParam represents Net.ApiUserTrainingCodeItemParam (Type 14177).
type ApiUserTrainingCodeItemParam struct {
	Idx int `json:"idx"`
	I   int `json:"i"`
}

// ApiUserTrainingCodeParam represents Net.ApiUserTrainingCodeParam (Type 14176).
type ApiUserTrainingCodeParam struct {
	Ver   uint32                         `json:"ver"`
	Tnam  string                         `json:"tnam"`
	Tlng  int                            `json:"tlng"`
	Unam  string                         `json:"unam"`
	Ulng  int                            `json:"ulng"`
	Trico string                         `json:"trico"`
	Mem   []ApiUserPokemonParam          `json:"mem"`
	Itms  []ApiUserTrainingCodeItemParam `json:"itms"`
	Fav   byte                           `json:"fav"`
	CreAt uint64                         `json:"creAt"`
}

// ApiRateParam represents Net.ApiRateParam (Type 13979).
type ApiRateParam struct {
	Ra *float32 `json:"ra,omitempty"`
}

// ApiTeamParam represents Net.ApiTeamParam (Type 13978) used in battle match/resume.
type ApiTeamParam struct {
	Fri   string                `json:"fri"`
	Nam   string                `json:"nam"`
	Lng   int                   `json:"lng"`
	Cou   int                   `json:"cou"`
	Trtit string                `json:"trtit"`
	Trico string                `json:"trico"`
	Trbad []string              `json:"trbad"`
	Rk    int16                 `json:"rk"`
	Wic   int                   `json:"wic"`
	Rat   ApiRateParam          `json:"rat"`
	Par   int                   `json:"par"`
	Tme   []ApiUserPokemonParam `json:"tme"`
	Bamin int                   `json:"bamin"`
	Ava   ApiUserAvatarParam    `json:"ava"`
	Tse   []int                 `json:"tse"`
}

// SpecialRuleSetting represents Net.SpecialRuleSetting (Type 13613).
type SpecialRuleSetting struct {
	Typ int `json:"typ"`
	Use int `json:"use"`
}

// BattleStartMatchingPmp represents Net.ApiBattleStartMatchingResponseParam.UniqueParam (Type 13616).
type BattleStartMatchingPmp struct {
	Exp    int         `json:"exp"`
	Atk    string      `json:"atk"`
	Ehsh   string      `json:"ehsh"`
	Ws     string      `json:"ws"`
	Type   string      `json:"type"`
	Bamin  int         `json:"bamin"`
	Lmst   uint64      `json:"lmst"`
	Mawarn byte        `json:"mawarn"`
	Mast   uint64      `json:"mast"`
	UD     *ApiUdParam `json:"UD,omitempty"`
}

// BattleStartBattlePmp represents Net.ApiBattleStartBattleResponseParam.UniqueParam (Type 13611).
type BattleStartBattlePmp struct {
	Rmid   string               `json:"rmid"`
	Sprule []SpecialRuleSetting `json:"sprule"`
	Exp    *int                 `json:"exp,omitempty"`
	Atk    string               `json:"atk"`
	Ehsh   string               `json:"ehsh"`
	Ws     string               `json:"ws"`
	Type   string               `json:"type"`
	UD     *ApiUdParam          `json:"UD,omitempty"`
}

// ApiBattleEndUserResourceParam represents Net.ApiBattleEndUserResourceParam (Type 13586).
type ApiBattleEndUserResourceParam struct {
	Type int16 `json:"type"`
	Id   int   `json:"id"`
	Num  int   `json:"num"`
}

// ApiBattleEndBattleRewardParam represents Net.ApiBattleEndBattleRewardParam (Type 13585).
type ApiBattleEndBattleRewardParam struct {
	Reason   int                             `json:"reason"`
	Param    int                             `json:"param"`
	Resource []ApiBattleEndUserResourceParam `json:"resource"`
}

// BattleEndBattlePmp represents Net.ApiBattleEndBattleResponseParam.UniqueParam (Type 13583).
type BattleEndBattlePmp struct {
	Bca    byte                            `json:"bca"`
	Rul    byte                            `json:"rul"`
	Stt    int                             `json:"stt"`
	Pobf   int                             `json:"pobf"`
	Poaf   int                             `json:"poaf"`
	Rkbf   int16                           `json:"rkbf"`
	Rkaf   int16                           `json:"rkaf"`
	Mrkbf  int16                           `json:"mrkbf"`
	Mrkaf  int16                           `json:"mrkaf"`
	Wicbf  int                             `json:"wicbf"`
	Wicaf  int                             `json:"wicaf"`
	Bopo   int                             `json:"bopo"`
	Rabf   float32                         `json:"rabf"`
	Raaf   float32                         `json:"raaf"`
	Et     int                             `json:"et"`
	Rwd    []ApiBattleEndBattleRewardParam `json:"rwd"`
	Rank   *int                            `json:"rank,omitempty"`
	Vpbm   int                             `json:"vpbm"`
	Rptflg byte                            `json:"rptflg"`
	Regag  byte                            `json:"regag"`
	UD     *ApiUdParam                     `json:"UD,omitempty"`
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
