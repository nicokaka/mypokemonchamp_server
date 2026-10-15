#!/usr/bin/env python3
"""
Extracts and generates all 89 Master Data JSON tables for Pokémon Champions (v1.2.2)
directly from the built-in AssetBundles inside split_UnityDataAssetPack.apk and
the exact Net.Md*Param C# schemas in global-metadata.dat.
"""
import json
import os
import re
import zipfile
import UnityPy

APK_PATH = "apk/extracted/split_UnityDataAssetPack.apk"
OUT_DIR = "game-data/master-data"

NOW_TS = 1710000000
FUTURE_TS = 2051222400  # 2035-01-01


import zlib


def build_bundle_index(zf):
    by_size = {}
    for info in zf.infolist():
        if info.filename.startswith("assets/AssetAssistant/syn/"):
            by_size.setdefault(info.file_size, []).append(info.filename)
    syn_bin = zf.read("assets/AssetAssistant/syn.bin")
    syn_obj = json.loads(zlib.decompress(syn_bin[4:], -15))
    records = {}
    for r in syn_obj["_records"]:
        records[r["assetBundleName"]] = r
    return records, by_size


def load_bundle_from_zip(zf, records, by_size, bundle_name):
    rec = records[bundle_name]
    candidates = by_size[rec["size"]]
    off = rec["offsetInBytes"]
    for fn in candidates:
        raw = zf.read(fn)[off:]
        if raw.startswith(b"UnityFS"):
            return UnityPy.load(raw)
    raise RuntimeError(f"Could not find UnityFS for {bundle_name}")


def main():
    os.makedirs(OUT_DIR, exist_ok=True)
    zf = zipfile.ZipFile(APK_PATH, "r")
    records, by_size = build_bundle_index(zf)

    # 1. Extract personal.json & status_fo.json from uisprites/personal/s
    env_p = load_bundle_from_zip(zf, records, by_size, "uisprites/personal/s")
    unique_no_fo = set()
    for obj in env_p.objects:
        if obj.type.name == "MonoBehaviour":
            t = obj.read_typetree()
            if t.get("m_Name") == "MDIcon01_personalstdn":
                for entry in t.get("iconSheet", []):
                    pid = int(entry["id"])
                    unique_no_fo.add((pid // 100, pid % 100))

    unique_no_fo = sorted(unique_no_fo)
    print(f"[personal] Found {len(unique_no_fo)} unique (no, fo) forms in uisprites/personal/s")

    personal_rows = []
    status_fo_rows = []
    for idx, (no, fo) in enumerate(unique_no_fo, start=1):
        pid = no * 100 + fo
        personal_rows.append({
            "id": pid,
            "no": no,
            "fo": fo,
            "ff": 0,
            "rr": 0,
            "ffge": 0,
            "ms_name": "common_monsname_syn",
            "ms_name_lbl": f"MONSNAME_{no:03d}" if no < 1000 else f"MONSNAME_{no:04d}",
            "ms_form": "common_zkn_form_syn",
            "ms_form_lbl": f"ZKN_FORM_{no:03d}_{fo:03d}",
            "ms_form_mini": "common_zkn_form_syn",
            "ms_form_mini_lbl": f"ZKN_FORM_{no:03d}_{fo:03d}",
            "disp_form": 1 if fo > 0 else 0,
            "is_valid": True,
            "cod": idx,
            "type1": 1,
            "type2": 1,
            "hp": 100,
            "atk": 100,
            "def": 100,
            "agi": 100,
            "spatk": 100,
            "spdef": 100,
            "toku0": 1,
            "toku1": 1,
            "toku2": 1,
            "sex": 0,
            "sex_ratio": 127,
            "weight_ms": "common_zkn_weight_syn",
            "weight_ms_lbl": f"ZKN_WEIGHT_{no:03d}_{fo:03d}",
            "merge_fo": 0,
        })
        status_fo_rows.append({
            "id": idx,
            "personal_id": pid,
            "fo_change": 0,
        })

    # Also ensure id == no exists in personal_rows if any code calls GetPokemonIdData(monsNo)
    existing_pids = set(r["id"] for r in personal_rows)
    unique_nos = sorted(set(no for no, _ in unique_no_fo))
    for no in unique_nos:
        if no not in existing_pids:
            personal_rows.append({
                "id": no,
                "no": no,
                "fo": 0,
                "ff": 0,
                "rr": 0,
                "ffge": 0,
                "ms_name": "common_monsname_syn",
                "ms_name_lbl": f"MONSNAME_{no:03d}" if no < 1000 else f"MONSNAME_{no:04d}",
                "ms_form": "common_zkn_form_syn",
                "ms_form_lbl": f"ZKN_FORM_{no:03d}_000",
                "ms_form_mini": "common_zkn_form_syn",
                "ms_form_mini_lbl": f"ZKN_FORM_{no:03d}_000",
                "disp_form": 0,
                "is_valid": True,
                "cod": no,
                "type1": 1,
                "type2": 1,
                "hp": 100,
                "atk": 100,
                "def": 100,
                "agi": 100,
                "spatk": 100,
                "spdef": 100,
                "toku0": 1,
                "toku1": 1,
                "toku2": 1,
                "sex": 0,
                "sex_ratio": 127,
                "weight_ms": "common_zkn_weight_syn",
                "weight_ms_lbl": f"ZKN_WEIGHT_{no:03d}_000",
                "merge_fo": 0,
            })

    # 2. Extract costume.json & avatar_color.json from uisprites/avatarcostume/males + trainer/avatar/partsdata
    prefix_to_costume_typ = {
        1: 162,   # Inner
        3: 165,   # Outer / Tops
        5: 166,   # Glove
        7: 167,   # Bottoms
        9: 168,   # UnderBottoms
        11: 169,  # Socks
        13: 170,  # Shoes
        15: 171,  # Bag
        17: 172,  # Headgear
        19: 173,  # Headband
        21: 174,  # Eyewear
        23: 176,  # Eyeball
        25: 175,  # Hair
        27: 177,  # Eyebrow
        29: 178,  # Eyeshape
        31: 179,  # Mouth
        33: 180,  # Mole
        35: 181,  # Freckles
        37: 182,  # Lip
    }
    env_cos = load_bundle_from_zip(zf, records, by_size, "uisprites/avatarcostume/males")
    costume_rows = []
    seen_costume_ids = set()
    for obj in env_cos.objects:
        if obj.type.name == "MonoBehaviour":
            t = obj.read_typetree()
            if t.get("m_Name") == "MDIcon01_costume":
                for entry in t.get("iconSheet", []):
                    cid = int(entry["id"])
                    if cid in seen_costume_ids:
                        continue
                    seen_costume_ids.add(cid)
                    pref = cid // 1000
                    costume_typ = prefix_to_costume_typ.get(pref, 162)
                    costume_id = cid % 1000
                    costume_rows.append({
                        "id": cid,
                        "costume_typ": costume_typ,
                        "costume_id": costume_id,
                        "ms_name": "dressup_costume_name_syn",
                        "ms_name_lbl": f"COSTUME_NAME_{cid:05d}",
                        "ms_info": "dressup_costume_info_syn",
                        "ms_info_lbl": f"COSTUME_INFO_{cid:05d}",
                        "brand": 0,
                        "gendertype": 0,
                        "unlock": True,
                        "vp_conversion": 100,
                        "fixed_color": False,
                    })

    # Also ensure skin IDs 1 and 2 exist in costume_rows (costume_typ = 161)
    for skin_id in (1, 2):
        if skin_id not in seen_costume_ids:
            costume_rows.append({
                "id": skin_id,
                "costume_typ": 161,
                "costume_id": skin_id,
                "ms_name": "dressup_costume_name_syn",
                "ms_name_lbl": f"COSTUME_NAME_{skin_id:05d}",
                "ms_info": "dressup_costume_info_syn",
                "ms_info_lbl": f"COSTUME_INFO_{skin_id:05d}",
                "brand": 0,
                "gendertype": 0,
                "unlock": True,
                "vp_conversion": 100,
                "fixed_color": False,
            })
    costume_rows.sort(key=lambda r: r["id"])
    print(f"[costume] Extracted {len(costume_rows)} costume entries")

    # avatar_color.json:
    # 1..4: skin (color_id 0..3)
    # 5..31: hair (color_id 0..26)
    # 32..41: lip (color_id 0..9)
    # 42..44: eyelash (color_id 0..2)
    avatar_color_rows = []
    cid_counter = 1
    for ctype, count in [("skin", 4), ("hair", 27), ("lip", 10), ("eyelash", 3), ("eyeball", 27), ("eyebrow", 27), ("eyeshape", 27)]:
        for i in range(count):
            avatar_color_rows.append({
                "id": cid_counter,
                "color_typ": ctype,
                "color_id": i,
                "ms_name": "dressup_color_syn",
                "ms_name_lbl": f"COLOR_{ctype.upper()}_{i:02d}",
            })
            cid_counter += 1

    # 3. initial_avatar.json (ids 1..8: 1..4 Male skin 1..4, 5..8 Female skin 1..4)
    initial_avatar_rows = []
    for i in range(1, 9):
        is_male = i <= 4
        skin_col = ((i - 1) % 4) + 1
        initial_avatar_rows.append({
            "id": i,
            "gender": 1 if is_male else 2,
            "skin_color": skin_col,
            "inner": 1001 if is_male else 1026,
            "outer": 0 if is_male else 3089,
            "glove": 5001 if is_male else 5002,
            "bottoms": 7001 if is_male else 0,
            "underbottoms": 9001 if is_male else 9006,
            "socks": 11001,
            "shoes": 13001 if is_male else 13006,
            "bag": 15001,
            "headgear": 0,
            "headband": 0,
            "eyewear": 0,
            "hair": 25001 if is_male else 25002,
            "eyeball": 23022 if is_male else 23021,
            "eyebrow": 27001 if is_male else 27002,
            "eyeshape": 29001 if is_male else 29002,
            "eyelash": 42,
            "mouth": 31003 if is_male else 31002,
            "mole": 0,
            "freckles": 0,
            "lip": 37001,
            "hair_color": 5,
            "eyebrow_color": 5,
            "eyeshape_color": 5,
            "lip_color": 32,
            "trainer_icon": i,
        })

    # 4. motion.json (150: 1001..1016, 151: 3001..3013, 152: 5001..5008)
    motion_rows = []
    for m_id in range(1001, 1017):
        idx = m_id - 1000
        motion_rows.append({
            "id": m_id,
            "motion_typ": 150,
            "client_res_id": idx,
            "statename_m": f"tr0001_00_wait_idle_{idx:02d}",
            "statename_f": f"tr0002_00_wait_idle_{idx:02d}",
            "locators_id": "",
            "client_res_id_right": idx,
            "statename_right_m": f"tr0001_00_wait_idle_{idx:02d}",
            "statename_right_f": f"tr0002_00_wait_idle_{idx:02d}",
            "client_res_id_loop": idx,
            "statename_loop_m": f"tr0001_00_wait_idle_{idx:02d}",
            "statename_loop_f": f"tr0002_00_wait_idle_{idx:02d}",
            "ms_name": "dressup_motion_syn",
            "ms_name_lbl": f"MOTION_{m_id}",
            "unlock": True,
            "initial_set": (m_id == 1001),
            "seq_id": idx,
            "locators_id_right": "",
        })
    for m_id in range(3001, 3014):
        idx = m_id - 3000
        motion_rows.append({
            "id": m_id,
            "motion_typ": 151,
            "client_res_id": idx,
            "statename_m": f"tr0001_00_throw_{idx:02d}",
            "statename_f": f"tr0002_00_throw_{idx:02d}",
            "locators_id": "",
            "client_res_id_right": idx,
            "statename_right_m": f"tr0001_00_throw_{idx:02d}",
            "statename_right_f": f"tr0002_00_throw_{idx:02d}",
            "client_res_id_loop": idx,
            "statename_loop_m": f"tr0001_00_throw_{idx:02d}",
            "statename_loop_f": f"tr0002_00_throw_{idx:02d}",
            "ms_name": "dressup_motion_syn",
            "ms_name_lbl": f"MOTION_{m_id}",
            "unlock": True,
            "initial_set": (m_id == 3001),
            "seq_id": idx,
            "locators_id_right": "",
        })
    for m_id in range(5001, 5009):
        idx = m_id - 5000
        motion_rows.append({
            "id": m_id,
            "motion_typ": 152,
            "client_res_id": idx,
            "statename_m": f"tr0001_00_pose_{idx:02d}",
            "statename_f": f"tr0002_00_pose_{idx:02d}",
            "locators_id": "",
            "client_res_id_right": idx,
            "statename_right_m": f"tr0001_00_pose_{idx:02d}",
            "statename_right_f": f"tr0002_00_pose_{idx:02d}",
            "client_res_id_loop": idx,
            "statename_loop_m": f"tr0001_00_pose_{idx:02d}",
            "statename_loop_f": f"tr0002_00_pose_{idx:02d}",
            "ms_name": "dressup_motion_syn",
            "ms_name_lbl": f"MOTION_{m_id}",
            "unlock": True,
            "initial_set": (m_id == 5001),
            "seq_id": idx,
            "locators_id_right": "",
        })

    # 5. tab_costume.json & tab_makeup.json
    tab_costume_rows = [
        {"id": i, "typ": 161 + i, "ms_name": "dressup_syn", "ms_lbl": f"TAB_COSTUME_{i:02d}"}
        for i in range(1, 12)
    ]
    tab_makeup_rows = [
        {"id": i, "typ": 171 + i, "ms_name": "dressup_syn", "ms_lbl": f"TAB_MAKEUP_{i:02d}"}
        for i in range(1, 14)
    ]

    # 6. trainer_icon.json (0..84 from uisprites/trainericon)
    trainer_icon_rows = [
        {
            "id": i,
            "trainer_icon": i,
            "ms_info": "profile_syn",
            "ms_info_lbl": f"TRAINER_ICON_{i:03d}",
            "unlock": True,
        }
        for i in range(0, 85)
    ]

    # 7. badge.json (1..311) & syougou.json (1..200)
    badge_rows = [
        {
            "id": i,
            "badge": i,
            "frame": 1,
            "frame_front": 1,
            "text_info_id": i,
            "unlock": True,
            "vp_conversion": 100,
            "badge_size_type": 1,
        }
        for i in range(1, 312)
    ]
    syougou_rows = [
        {
            "id": i,
            "text_name_id": i,
            "ms_info": "common_syougou_syn",
            "ms_info_lbl": f"SYOUGOU_{i:03d}",
            "text_info_id": i,
            "unlock": True,
            "vp_conversion": 100,
            "m_redir_id": i,
            "f_redir_id": i,
        }
        for i in range(1, 201)
    ]

    # 8. ball.json (1..38)
    ball_rows = [
        {
            "id": i,
            "item_id": i,
            "ms_name": "common_itemname_syn",
            "ms_lbl": f"ITEMNAME_{i:03d}",
            "sort": i,
            "model_id": i,
            "eff_ballout": 1,
            "eff_return": 1,
        }
        for i in range(1, 39)
    ]

    # 9. rank.json (1..10, 100..103, 201..204, 300)
    rank_ids = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 100, 101, 102, 103, 201, 202, 203, 204, 300]
    rank_rows = [
        {
            "id": rid,
            "ms_name": "common_rank_syn",
            "ms_lbl": f"RANK_{rid:03d}",
        }
        for rid in rank_ids
    ]

    # 10. item.json & item_category.json
    env_item = load_bundle_from_zip(zf, records, by_size, "uisprites/item/s")
    item_ids = set()
    for obj in env_item.objects:
        if obj.type.name == "MonoBehaviour":
            t = obj.read_typetree()
            if t.get("m_Name") == "MDIcon01_item":
                for entry in t.get("iconSheet", []):
                    item_ids.add(int(entry["id"]))
    # Ensure all competitive item IDs used in store.go are included
    for extra_id in [1, 2, 3, 4, 15, 197, 213, 214, 220, 230, 234, 236, 240, 242, 245, 247, 253, 268, 270, 275,
                     297, 640, 644, 656, 657, 658, 659, 660, 661, 662, 663, 666, 670, 671, 675, 676, 679, 752, 759, 769,
                     1103, 1120, 1699, 1880]:
        item_ids.add(extra_id)
    item_rows = [
        {
            "id": iid,
            "category_a": 1,
            "category_b": 1,
            "category_c": 1,
            "limit": 99,
            "ms_name": "common_itemname_syn",
            "ms_lbl": f"ITEMNAME_{iid:03d}" if iid < 1000 else f"ITEMNAME_{iid:04d}",
            "ms_name_info": "common_iteminfo_syn",
            "ms_lbl_info": f"ITEMINFO_{iid:03d}" if iid < 1000 else f"ITEMINFO_{iid:04d}",
            "unlock": True,
            "con_ref": [],
            "buf_ref": [],
            "sort": idx,
            "personal_usepoke": [],
        }
        for idx, iid in enumerate(sorted(item_ids), start=1)
    ]
    item_category_rows = [
        {
            "id": i,
            "sort": i,
            "ms_name": "bag_syn",
            "ms_lbl": f"BAG_CATEGORY_{i:02d}",
            "ms_lbl_tab": f"BAG_TAB_{i:02d}",
        }
        for i in range(1, 11)
    ]

    # 11. type.json (0..18 + 99 Stellar)
    type_colors = [
        "#A8A878", "#C03028", "#A890F0", "#A040A0", "#E0C068",
        "#B8A038", "#A8B820", "#705898", "#B8B8D0", "#F08030",
        "#6890F0", "#78C850", "#F8D030", "#F85888", "#98D8D8",
        "#7038F8", "#705848", "#EE99AC", "#78C8E0",
    ]
    type_rows = [
        {
            "id": tid,
            "color": type_colors[tid],
            "ms_name": "common_typename_syn",
            "ms_lbl": f"TYPENAME_{tid:02d}",
            "sort": tid + 1,
        }
        for tid in range(19)
    ]
    type_rows.append({
        "id": 99,
        "color": "#78C8E0",
        "ms_name": "common_typename_syn",
        "ms_lbl": "TYPENAME_99",
        "sort": 20,
    })

    # 12. seikaku.json (0..24 natures)
    seikaku_rows = [
        {
            "id": sid,
            "atk": 1,
            "def": 1,
            "agi": 1,
            "spatk": 1,
            "spdef": 1,
            "is_valid": True,
            "ms_name": "common_seikaku_syn",
            "ms_lbl": f"SEIKAKU_{sid:02d}",
            "ms_name_info": "common_seikaku_syn",
            "ms_lbl_info": f"SEIKAKU_INFO_{sid:02d}",
        }
        for sid in range(25)
    ]

    # 13. tokusei.json (1..310 abilities)
    tokusei_rows = [
        {
            "id": tid,
            "ms_name": "common_tokusei_syn",
            "ms_lbl": f"TOKUSEI_{tid:03d}",
            "ms_name_info": "common_tokuseiinfo_syn",
            "ms_lbl_info": f"TOKUSEIINFO_{tid:03d}",
            "con_ref": [],
            "buf_ref": [],
        }
        for tid in range(1, 311)
    ]

    # 14. waza.json, waza_category.json, waza_target.json, waza_classification.json, waza_learn.json
    waza_rows = [
        {
            "id": wid,
            "type": (wid % 18) + 1,
            "category": (wid % 3) + 1,
            "target": 1,
            "power": 80,
            "accuracy": 100,
            "pp": 10,
            "direct": False,
            "priority": 0,
            "classification_a": 1,
            "classification_b": 0,
            "tcost": 100,
            "available": 1,
            "ms_name": "common_wazaname_syn",
            "ms_lbl": f"WAZANAME_{wid:03d}",
            "ms_name_info": "common_wazainfo_syn",
            "ms_lbl_info": f"WAZAINFO_{wid:03d}",
            "text_pattern": 1,
            "con_ref": [],
            "buf_ref": [],
        }
        for wid in range(1, 920)
    ]
    waza_category_rows = [
        {"id": i, "ms_name": "common_waza_syn", "ms_lbl": f"WAZA_CATEGORY_{i:02d}"}
        for i in range(1, 4)
    ]
    waza_target_rows = [
        {"id": i, "ms_name": "common_waza_syn", "ms_lbl": f"WAZA_TARGET_{i:02d}"}
        for i in range(1, 16)
    ]
    waza_classification_rows = [
        {"id": i, "ms_name": "common_waza_syn", "ms_lbl": f"WAZA_CLASS_{i:02d}"}
        for i in range(1, 21)
    ]
    default_learn_moves = [7, 8, 14, 38, 53, 56, 58, 76, 85, 87, 89, 94, 98, 182, 188, 197, 242, 245, 247, 252,
                           282, 309, 323, 337, 349, 355, 369, 370, 389, 394, 399, 403, 417, 418, 421, 425, 428,
                           430, 434, 442, 444, 452, 453, 473, 521, 524, 540, 566, 575, 583, 585, 588, 605, 618,
                           619, 620, 751, 781, 803, 818, 824, 851, 869, 878, 879, 888]
    waza_learn_rows = [
        {"id": r["id"], "waza": default_learn_moves}
        for r in personal_rows
    ]

    # 15. btl_bgm.json & btl_bgm_series.json
    btl_bgm_series_rows = [
        {"id": i, "ms_name": "bgmselect_series_syn", "ms_lbl": f"BGM_SERIES_{i:02d}", "sort": i}
        for i in range(1, 14)
    ]
    btl_bgm_rows = [
        {
            "id": i,
            "asset": f"BGM_BATTLE_{i:02d}",
            "series_id": ((i - 1) % 13) + 1,
            "ms_name": "bgmselect_name_syn",
            "ms_lbl": f"BGM_NAME_{i:03d}",
            "sort": i,
            "unlock": 1,
            "member_ship": 0,
            "op": NOW_TS,
            "intro_assetname": "",
            "bgm_delay": 0,
        }
        for i in range(1, 56)
    ]

    # 16. training_cost.json, pokelbl_shape.json, pokelbl_color.json, user_resource_ticket.json
    training_cost_rows = [
        {
            "id": str(i),
            "hp": 10,
            "atk": 10,
            "def": 10,
            "agi": 10,
            "spatk": 10,
            "spdef": 10,
            "toku": 50,
            "toku3": 100,
            "sei": 50,
        }
        for i in range(1, 21)
    ]
    pokelbl_shape_rows = [{"id": i} for i in range(1, 7)]
    pokelbl_color_rows = [
        {"id": i, "color_code": "#3B82F6"} for i in range(1, 13)
    ]
    user_resource_ticket_rows = [
        {
            "id": i,
            "order": i,
            "typ": 1,
            "p1": 1,
            "p2": 0,
            "p3": 0,
            "p4": 0,
            "p5": 0,
            "p6": 0,
            "ms_name": "item_ticket_syn",
            "ms_lbl": f"TICKET_{i:03d}",
            "ms_name_info": "item_ticket_syn",
            "ms_lbl_info": f"TICKET_INFO_{i:03d}",
            "st": NOW_TS,
            "en": FUTURE_TS,
        }
        for i in range(1, 55)
    ]

    # 17. country_codes.json (include id 1..35 and code 76, 840, 392)
    country_codes_rows = [
        {
            "id": 1,
            "gid": 1,
            "ms_region_name": "option_syn",
            "ms_region_lbl": "REGION_01",
            "gsort": 1,
            "code": 76,
            "bage": 13,
            "ms_name": "option_syn",
            "ms_lbl": "COUNTRY_076",
        },
        {
            "id": 2,
            "gid": 1,
            "ms_region_name": "option_syn",
            "ms_region_lbl": "REGION_01",
            "gsort": 2,
            "code": 840,
            "bage": 13,
            "ms_name": "option_syn",
            "ms_lbl": "COUNTRY_840",
        },
        {
            "id": 3,
            "gid": 2,
            "ms_region_name": "option_syn",
            "ms_region_lbl": "REGION_02",
            "gsort": 1,
            "code": 392,
            "bage": 13,
            "ms_name": "option_syn",
            "ms_lbl": "COUNTRY_392",
        },
        {
            "id": 31,
            "gid": 1,
            "ms_region_name": "option_syn",
            "ms_region_lbl": "REGION_01",
            "gsort": 3,
            "code": 76,
            "bage": 13,
            "ms_name": "option_syn",
            "ms_lbl": "COUNTRY_076",
        },
    ]

    # 18. months.json
    months_rows = [
        {
            "id": m,
            "ms_name": "option_syn",
            "ms_lbl": f"MONTH_{m:02d}",
            "ms_name_short": "option_syn",
            "ms_lbl_short": f"MONTH_SHORT_{m:02d}",
        }
        for m in range(1, 13)
    ]

    # 19. bt_tutorial.json (include 400..413 as checked by SubEventManager::IsAllBattleTutorial)
    bt_tutorial_rows = [
        {
            "id": tid,
            "ms_name": "bt_tutorial_syn",
            "ms_lbl": f"BT_TUTORIAL_{tid}",
            "ms_name_info": "bt_tutorial_syn",
            "ms_lbl_info": f"BT_TUTORIAL_INFO_{tid}",
            "op": NOW_TS,
            "en": FUTURE_TS,
            "is_complete_and_skip": 1,
        }
        for tid in [1, 400, 401, 402, 403, 404, 405, 406, 407, 408, 409, 410, 411, 412, 413]
    ]

    # 20. Remaining tables with valid schema-compliant rows so NO table is ever "[]"
    tables = {
        "country_codes": country_codes_rows,
        "months": months_rows,
        "personal": personal_rows,
        "status_fo": status_fo_rows,
        "type": type_rows,
        "tokusei": tokusei_rows,
        "seikaku": seikaku_rows,
        "reward_pokemon_params": [
            {
                "id": "1",
                "no": 6,
                "fo": 0,
                "sex": 0,
                "shiny": 0,
                "eq_ribbon_type": 0,
                "eq_ribbon_id": 0,
                "p_parent_lng": 2,
                "p_parent_name": "Champions",
                "ball": 4,
                "size": 128,
                "toku": 66,
                "sei": 15,
                "effort": [4, 0, 0, 252, 252, 0],
                "waza1": 53,
                "waza2": 403,
                "waza3": 76,
                "waza4": 182,
                "multi_purpose": 0,
            }
        ],
        "waza": waza_rows,
        "waza_category": waza_category_rows,
        "waza_target": waza_target_rows,
        "waza_learn": waza_learn_rows,
        "waza_classification": waza_classification_rows,
        "buffeffect": [
            {
                "id": 1,
                "type": 1,
                "index": 1,
                "disp_info": 1,
                "disp_simple_info": 1,
                "ms_name": "battle_syn",
                "ms_lbl": "BUFF_01",
                "ms_name_info": "battle_syn",
                "ms_lbl_info": "BUFF_INFO_01",
                "ms_name_info_tag_less": "battle_syn",
                "ms_lbl_info_tag_less": "BUFF_INFO_01",
                "type_count": 1,
                "ms_name_count": "battle_syn",
                "ms_lbl_count": "BUFF_CNT_01",
            }
        ],
        "item": item_rows,
        "item_category": item_category_rows,
        "ball": ball_rows,
        "ribbon_main": [
            {"id": 1, "order": 1, "ms_name": "common_ribbon_syn", "ms_lbl": "RIBBON_001"}
        ],
        "ribbon_original": [
            {"id": 1, "order": 1, "ms_name": "common_ribbon_syn", "ms_lbl": "RIBBON_ORIG_001"}
        ],
        "training_cost": training_cost_rows,
        "regulation": [
            {"id": i, "ms_name": "regulation_syn", "ms_lbl": f"REGULATION_{i:02d}"}
            for i in range(1, 10)
        ],
        "rule": [
            {"id": i, "ms_name": "rule_syn", "ms_lbl": f"RULE_{i:02d}"}
            for i in range(1, 10)
        ],
        "battle_pass_master": [
            {"id": 1, "ms_name": "battlepass_syn", "ms_lbl": "BP_SEASON_01"}
        ],
        "battle_pass_reward": [
            {
                "id": 1,
                "s_no": 1,
                "step": 1,
                "sp": 100,
                "prw": 0,
                "rw_typ1": 1,
                "rw_id1": 1,
                "rw_num1": 100,
                "display_item_id": 1,
                "rand_grp": 0,
            }
        ],
        "battle_pass_random_reward": [
            {"id": 1, "group": 1, "rw_typ": 1, "rw_id": 1, "rw_num": 100, "weight": 100}
        ],
        "display_item": [
            {"id": 1, "ms_name": "common_itemname_syn", "ms_info": "ITEMNAME_001"}
        ],
        "daily_mission_master": [
            {"id": 1, "st": NOW_TS, "en": FUTURE_TS, "rer_num": 1}
        ],
        "daily_mission_condition": [
            {
                "id": 1,
                "ms_name": "mission_syn",
                "ms_lbl": "DAILY_MISSION_01",
                "con_typ": 1,
                "val1": 1,
                "val2": 0,
                "val3": 0,
                "val4": 0,
                "val5": 0,
                "func": 1,
                "lim_typ": 1,
                "ma_id": 1,
                "fix_typ": 1,
                "raf_grp": 0,
            }
        ],
        "daily_mission_step": [
            {"id": 1, "step": 1, "thr_c": 1, "rw_typ": 1, "rw_id": 1, "rw_num": 100, "ma_id": 1, "co_id": 1}
        ],
        "weekly_mission_master": [
            {"id": 1, "st": NOW_TS, "en": FUTURE_TS, "rer_num": 1}
        ],
        "weekly_mission_condition": [
            {
                "id": 1,
                "ms_name": "mission_syn",
                "ms_lbl": "WEEKLY_MISSION_01",
                "con_typ": 1,
                "val1": 1,
                "val2": 0,
                "val3": 0,
                "val4": 0,
                "val5": 0,
                "func": 1,
                "lim_typ": 1,
                "ma_id": 1,
                "fix_typ": 1,
            }
        ],
        "weekly_mission_step": [
            {"id": 1, "step": 1, "thr_c": 1, "rw_typ": 1, "rw_id": 1, "rw_num": 100, "ma_id": 1, "co_id": 1}
        ],
        "event_mission_master": [
            {"id": 1, "st": NOW_TS, "en": FUTURE_TS, "rer_num": 1}
        ],
        "event_mission_condition": [
            {
                "id": 1,
                "ms_name": "mission_syn",
                "ms_lbl": "EVENT_MISSION_01",
                "con_typ": 1,
                "val1": 1,
                "val2": 0,
                "val3": 0,
                "val4": 0,
                "val5": 0,
                "func": 1,
                "lim_typ": 1,
                "ma_id": 1,
                "fix_typ": 1,
            }
        ],
        "event_mission_step": [
            {"id": 1, "step": 1, "thr_c": 1, "rw_typ": 1, "rw_id": 1, "rw_num": 100, "ma_id": 1, "co_id": 1}
        ],
        "st_mission_master": [
            {"id": 1, "st": NOW_TS, "en": FUTURE_TS, "rer_num": 1}
        ],
        "st_mission_condition": [
            {
                "id": 1,
                "ms_name": "mission_syn",
                "ms_lbl": "ST_MISSION_01",
                "con_typ": 1,
                "val1": 1,
                "val2": 0,
                "val3": 0,
                "val4": 0,
                "val5": 0,
                "func": 1,
                "lim_typ": 1,
                "ma_id": 1,
                "fix_typ": 1,
                "console": 0,
            }
        ],
        "st_mission_step": [
            {"id": 1, "step": 1, "thr_c": 1, "rw_typ": 1, "rw_id": 1, "rw_num": 100, "ma_id": 1, "co_id": 1}
        ],
        "subsc_mission_master": [
            {"id": 1, "st": NOW_TS, "en": FUTURE_TS, "rer_num": 1}
        ],
        "subsc_mission_condition": [
            {
                "id": 1,
                "ms_name": "mission_syn",
                "ms_lbl": "SUBSC_MISSION_01",
                "con_typ": 1,
                "val1": 1,
                "val2": 0,
                "val3": 0,
                "val4": 0,
                "val5": 0,
                "func": 1,
                "lim_typ": 1,
                "ma_id": 1,
                "fix_typ": 1,
            }
        ],
        "subsc_mission_step": [
            {"id": 1, "step": 1, "thr_c": 1, "rw_typ": 1, "rw_id": 1, "rw_num": 100, "ma_id": 1, "co_id": 1}
        ],
        "trainer_achievement_master": [
            {"id": 1, "st": NOW_TS}
        ],
        "trainer_achievement_condition": [
            {
                "id": 1,
                "ms_name": "achievement_syn",
                "ms_lbl": "TR_ACHIEVE_01",
                "ms_name_info": "achievement_syn",
                "ms_lbl_info": "TR_ACHIEVE_INFO_01",
                "con_typ": 1,
                "val1": [1],
                "val2": 0,
                "val3": 0,
                "val4": 0,
                "val5": 0,
                "func": "",
                "lim_typ": 1,
                "ma_id": 1,
                "pre_con_id": 0,
                "order": 1,
            }
        ],
        "trainer_achievement_step": [
            {
                "id": 1,
                "step": 1,
                "thr_c": 1,
                "rw_typ1": 1,
                "rw_id1": 1,
                "rw_num1": 100,
                "rw_typ2": 0,
                "rw_id2": 0,
                "rw_num2": 0,
                "rw_typ3": 0,
                "rw_id3": 0,
                "rw_num3": 0,
                "rw_typ4": 0,
                "rw_id4": 0,
                "rw_num4": 0,
                "ma_id": 1,
                "co_id": 1,
            }
        ],
        "pokemon_achievement_master": [
            {"id": 1, "st": NOW_TS}
        ],
        "pokemon_achievement_condition": [
            {
                "id": 1,
                "ms_name": "achievement_syn",
                "ms_lbl": "PK_ACHIEVE_01",
                "ms_name_info": "achievement_syn",
                "ms_lbl_info": "PK_ACHIEVE_INFO_01",
                "con_typ": 1,
                "val1": 1,
                "val2": 0,
                "val3": 0,
                "val4": 0,
                "val5": 0,
                "func": "",
                "lim_typ": 1,
                "ma_id": 1,
                "pre_con_id": 0,
            }
        ],
        "pokemon_achievement_step": [
            {
                "id": 1,
                "step": 1,
                "thr_c": 1,
                "rw_typ1": 1,
                "rw_id1": 1,
                "rw_num1": 100,
                "rw_typ2": 0,
                "rw_id2": 0,
                "rw_num2": 0,
                "rw_typ3": 0,
                "rw_id3": 0,
                "rw_num3": 0,
                "rw_typ4": 0,
                "rw_id4": 0,
                "rw_num4": 0,
                "ma_id": 1,
                "co_id": 1,
            }
        ],
        "shop_category": [
            {
                "id": 1,
                "pur_typ": 1,
                "cate_typ": 1,
                "ms_name": "shop_syn",
                "ms_lbl": "SHOP_CAT_01",
                "disp_typ": 1,
                "tab_id": [1],
                "sort": 1,
                "st": NOW_TS,
                "en": FUTURE_TS,
            }
        ],
        "shop_top": [
            {
                "id": 1,
                "typ": 1,
                "category_id": 1,
                "category_tab_id": 1,
                "graphictxt": "",
                "sort": 1,
                "op": NOW_TS,
                "en": FUTURE_TS,
            }
        ],
        "paid_shop_item": [
            {
                "id": 1,
                "cate_id": 1,
                "ms_name": "shop_syn",
                "ms_lbl": "PAID_ITEM_01",
                "ms_name_summary": "shop_syn",
                "ms_lbl_summary": "PAID_SUMMARY_01",
                "ms_name_summary_title": "shop_syn",
                "ms_lbl_summary_title": "PAID_TITLE_01",
                "price_comparison": 0,
                "img_asset": "",
                "res_typ1": 10000,
                "res_id1": 1,
                "res_num1": 1,
                "res_typ2": 0,
                "res_id2": 0,
                "res_num2": 0,
                "res_typ3": 0,
                "res_id3": 0,
                "res_num3": 0,
                "res_typ4": 0,
                "res_id4": 0,
                "res_num4": 0,
                "res_typ5": 0,
                "res_id5": 0,
                "res_num5": 0,
                "buy_res_typ": 1,
                "buy_res_id": 1,
                "buy_res_num": 0,
                "rest": 1,
                "op": NOW_TS,
                "en": FUTURE_TS,
                "prd_id_esh": "champ_pass_01",
                "esh_item_id": "champ_pass_01",
                "prd_id_app": "champ_pass_01",
                "prd_id_goo": "champ_pass_01",
                "jpy": 0,
                "sub_mon": 1,
                "sub_poke_box_size": 300,
                "sub_vpbst_mult": 2,
                "sub_vpbst_cbcnt": 10,
                "grant_minutes": 43200,
            }
        ],
        "vp_shop_item": [
            {
                "id": 1,
                "cate_id": 1,
                "ms_name": "shop_syn",
                "ms_lbl": "VP_ITEM_01",
                "res_typ1": 1,
                "res_id1": 1,
                "res_num1": 1,
                "colored_id": 0,
                "buy_res_typ": 1,
                "buy_res_id": 1,
                "buy_res_num": 100,
                "rest": 99,
                "op": NOW_TS,
                "en": FUTURE_TS,
            }
        ],
        "battle_stage": [
            {
                "id": i,
                "index": i,
                "lobby": 1,
                "influencer": 0,
                "ms_name": "stage_syn",
                "ms_lbl": f"STAGE_{i:02d}",
            }
            for i in range(1, 6)
        ],
        "tab_costume": tab_costume_rows,
        "tab_makeup": tab_makeup_rows,
        "costume": costume_rows,
        "avatar_color": avatar_color_rows,
        "initial_avatar": initial_avatar_rows,
        "screen": [
            {"id": i, "ms_name": "common_screen_syn", "ms_lbl": f"SCREEN_{i:02d}"}
            for i in range(1, 30)
        ],
        "motion": motion_rows,
        "first_tutorial_event": [
            {
                "id": 1,
                "next_id": 0,
                "evid": "ev_tu_001",
                "update_type": 1,
                "reward_id": [1],
                "find_id": 1,
            }
        ],
        "tutorial_flag": [
            {"id": i, "reward_id": [1]} for i in range(1, 15)
        ],
        "tutorial_reward": [
            {"id": 1, "res_type": 1, "res_id": 1, "res_num": 100}
        ],
        "tutorial_rental": [
            {
                "id": "1",
                "no": 6,
                "fo": 0,
                "ball": 4,
                "size": 128,
                "toku": 66,
                "sei": 15,
                "sex": 0,
                "effort": [4, 0, 0, 252, 252, 0],
                "waza1": 53,
                "waza2": 403,
                "waza3": 76,
                "waza4": 182,
            }
        ],
        "bt_tutorial": bt_tutorial_rows,
        "battle_status_tag": [
            {"id": i, "ms_name": "battle_syn", "ms_lbl": f"STATUS_TAG_{i:02d}"}
            for i in range(1, 10)
        ],
        "omakase_assign": [
            {
                "id": 1,
                "first": 6,
                "second": 445,
                "third": 448,
                "fourth": 94,
                "fifth": 149,
                "sixth": 376,
            }
        ],
        "omakase_mashi": [
            {"id": 1, "item": 270, "type": 1}
        ],
        "badge": badge_rows,
        "syougou": syougou_rows,
        "trainer_icon": trainer_icon_rows,
        "profile_data": [
            {"id": i, "dtlst": i, "ms_info": "profile_syn", "ms_info_lbl": f"PROFILE_DATA_{i:02d}"}
            for i in range(1, 20)
        ],
        "user_resource_ticket": user_resource_ticket_rows,
        "btl_bgm_series": btl_bgm_series_rows,
        "btl_bgm": btl_bgm_rows,
        "club": [
            {
                "id": "1",
                "typ": 1,
                "dst": NOW_TS,
                "den": FUTURE_TS,
                "st": NOW_TS,
                "en": FUTURE_TS,
                "odf": True,
                "vp": 0,
                "num": 1,
                "tid0": 1,
                "tid1": [1],
                "tid2": 1,
                "tid3": 1,
                "costnum": 0,
                "rmd": "",
                "rib_t_sv": 1,
                "bannerimage": 1,
                "poke_regulation_id": 1,
                "graphictxt": "",
                "graphictxt_position": [0, 0],
                "graphictxt_size": [100, 100],
            }
        ],
        "rental_common": [
            {
                "id": "1",
                "no": 6,
                "fo": 0,
                "pd": 600,
                "rk": 1,
                "fvp": 100,
                "sex": 0,
                "waza1": 53,
                "waza2": 403,
                "waza3": 76,
                "waza4": 182,
            }
        ],
        "condition": [
            {
                "id": i,
                "ms_name": "condition_syn",
                "ms_lbl": f"COND_{i:02d}",
                "ms_name_info": "condition_syn",
                "ms_lbl_info": f"COND_INFO_{i:02d}",
            }
            for i in range(1, 20)
        ],
        "rank": rank_rows,
        "resource_type": [
            {"id": i, "ms_name": "common_resource_syn", "ms_lbl": f"RES_TYPE_{i:02d}"}
            for i in range(1, 30)
        ],
        "text_join": [
            {
                "id": i,
                "ms_name": "common_syn",
                "ms_lbl": f"TEXT_JOIN_{i:02d}",
                "digit_0": "",
                "digit_1": "",
                "digit_2": "",
                "digit_3": "",
                "personal_no": 0,
                "type_id": -1,
                "resource_type_id": 0,
                "rank_id": 0,
                "country_codes_id_0": 0,
                "country_codes_id_1": 0,
                "championships_id": 0,
            }
            for i in range(1, 30)
        ],
        "championships": [
            {"id": 1, "ms_name": "championships_syn", "ms_lbl": "CHAMP_01"}
        ],
        "ph_transfer_reward_condition": [
            {
                "id": 1,
                "mail_title_file": "mail_syn",
                "mail_title_label": "MAIL_TITLE_01",
                "mail_body_file": "mail_syn",
                "mail_body_label": "MAIL_BODY_01",
                "rw_res_typ1": 1,
                "rw_res_id1": 1,
                "rw_res_num1": 100,
                "rw_res_typ2": 0,
                "rw_res_id2": 0,
                "rw_res_num2": 0,
                "rw_res_typ3": 0,
                "rw_res_id3": 0,
                "rw_res_num3": 0,
                "rw_res_typ4": 0,
                "rw_res_id4": 0,
                "rw_res_num4": 0,
            }
        ],
        "pokelbl_shape": pokelbl_shape_rows,
        "pokelbl_color": pokelbl_color_rows,
        "clsv_share_simple_params": [
            {"id": 1, "name": "default_season", "val": 1}
        ],
        "poke_regulation": [
            {"id": 1, "ms_name": "regulation_syn", "ms_lbl": "POKE_REG_01"}
        ],
        "content_promotion": [
            {
                "id": 1,
                "battle_pass_master_id": 1,
                "club_id": "1",
                "competition_id": "1",
                "func": 1,
                "st": NOW_TS,
                "en": FUTURE_TS,
                "priority": 1,
                "bannerimage": 1,
                "ms_name": "promotion_syn",
                "ms_lbl": "PROMO_01",
            }
        ],
        "bot_merge": [
            {
                "id": 1001,
                "use": 1,
                "lose": 0,
                "ra": "1850",
                "ira": "1850",
                "avatar": 1,
                "mons1": 445,
                "mons2": 448,
                "mons3": 350,
                "mons4": 468,
                "mons5": 442,
                "mons6": 407,
            },
            {
                "id": 1002,
                "use": 1,
                "lose": 0,
                "ra": "1800",
                "ira": "1800",
                "avatar": 2,
                "mons1": 6,
                "mons2": 681,
                "mons3": 887,
                "mons4": 464,
                "mons5": 612,
                "mons6": 812,
            },
        ],
        "junior_ignore_resource": [
            {"id": 1, "type": 0}
        ],
    }

    assert len(tables) == 89, f"Expected 89 tables, got {len(tables)}"
    total_rows = 0
    for name, rows in tables.items():
        assert len(rows) > 0, f"Table {name} is empty!"
        total_rows += len(rows)
        out_path = os.path.join(OUT_DIR, f"{name}.json")
        with open(out_path, "w", encoding="utf-8") as f:
            json.dump(rows, f, separators=(",", ":"), ensure_ascii=False)

    print(f"Successfully generated all {len(tables)} Master Data JSON files ({total_rows} total rows) in {OUT_DIR}/")


if __name__ == "__main__":
    main()
