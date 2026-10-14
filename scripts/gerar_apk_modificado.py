#!/usr/bin/env python3
"""
Patcher Automatizado do APK Pokémon Champions (v1.2.2 Build 3859)
- Altera o endpoint padrão em global-metadata.dat (StringLiteral[13153]) de:
    https://api.app.pokemonchampions.jp (35 bytes)
  para:
    https://164.152.40.114:9443 (27 bytes, atualizado na tabela de comprimentos IL2CPP v31)
- Libera o Regex de validação de URL em Net.NetEnvironment::SetApiEndpoint (StringLiteral[10175] e [10176])
- Aplica patch opcional em libil2cpp.so (0x451d1e0 e 0x451d23c) para envio de JSON puro em pmc
"""
import argparse
import hashlib
import shutil
import struct
import zipfile
import zlib
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parent.parent
DEFAULT_EXTRACTED_DIR = ROOT_DIR / "apk" / "extracted"
OUT_DIR = ROOT_DIR / "build_apk"


def patch_classes_dex(dex_bytes: bytes) -> bytes:
    buf = bytearray(dex_bytes)
    # Bypass Google Play Pairip LicenseClient in classes.dex:
    # 1. 0x445394: LicenseClient.checkLicense(Context)V -> return-void (0e 00)
    # 2. 0x445704: LicenseClient.initializeLicenseCheck()V -> return-void (0e 00)
    # 3. 0x444f1c: LicenseClient.isLocalCheckPassed()Z -> const/4 v0, 1; return v0 (12 10 0f 00)
    buf[0x445394:0x445396] = b"\x0e\x00"
    buf[0x445704:0x445706] = b"\x0e\x00"
    buf[0x444F1C:0x444F20] = b"\x12\x10\x0f\x00"

    # Recompute DEX SHA-1 signature (bytes 12..32 over bytes 32..EOF)
    buf[12:32] = hashlib.sha1(buf[32:]).digest()
    # Recompute DEX Adler32 checksum (bytes 8..12 over bytes 12..EOF)
    checksum = zlib.adler32(buf[12:]) & 0xFFFFFFFF
    struct.pack_into("<I", buf, 8, checksum)
    print(f"  [OK] classes.dex: Bypass LicenseClient aplicado (Adler32=0x{checksum:08x})")
    return bytes(buf)


def patch_global_metadata(md_bytes: bytes, target_url: str) -> bytes:
    buf = bytearray(md_bytes)
    hdr = struct.unpack_from("<66I", buf, 0)
    magic, version = hdr[0], hdr[1]
    if magic != 0xFAB11BAF or version != 31:
        raise ValueError(f"global-metadata.dat inválido: magic=0x{magic:x}, version={version}")

    sl_off = hdr[2]
    sld_off = hdr[4]

    def replace_literal(idx: int, new_str: str):
        old_len, data_idx = struct.unpack_from("<II", buf, sl_off + idx * 8)
        old_bytes = bytes(buf[sld_off + data_idx : sld_off + data_idx + old_len])
        new_bytes = new_str.encode("utf-8")
        if len(new_bytes) > old_len:
            raise ValueError(
                f"StringLiteral[{idx}] nova string ({len(new_bytes)}B) maior que original ({old_len}B: {old_bytes!r})"
            )
        # Atualiza o tamanho exato na tabela de StringLiteral (sem alterar offsets do arquivo!)
        struct.pack_into("<II", buf, sl_off + idx * 8, len(new_bytes), data_idx)
        # Sobrescreve os bytes e preenche o restante com zeros
        buf[sld_off + data_idx : sld_off + data_idx + old_len] = new_bytes.ljust(old_len, b"\x00")
        print(f"  [OK] StringLiteral[{idx}]: {old_bytes.decode('utf-8', 'ignore')!r} -> {new_str!r}")

    # 1. Endpoint principal: https://api.app.pokemonchampions.jp (max 35 chars)
    replace_literal(13153, target_url)
    # 2. Regex de validação de endpoint em Net.NetEnvironment::SetApiEndpoint
    replace_literal(10175, "^https?://.*$")
    replace_literal(10176, "^https?://.*$")

    return bytes(buf)


def patch_libil2cpp(so_bytes: bytes) -> bytes:
    buf = bytearray(so_bytes)
    # Em PT_LOAD[2], file_offset = va - 0x4000
    # 0x451d1e0: BL EncryptRequestPmpAuth -> MOV X0, X22 (0xaa1603e0)
    # 0x451d23c: BL EncryptRequestPmpApi  -> MOV X0, X22 (0xaa1603e0)
    patches = [
        (0x451D1E0 - 0x4000, 0x940069DC, 0xAA1603E0, "Bypass EncryptRequestPmpAuth -> Plain JSON"),
        (0x451D23C - 0x4000, 0x94006D21, 0xAA1603E0, "Bypass EncryptRequestPmpApi -> Plain JSON"),
    ]
    for off, expected, replacement, desc in patches:
        cur = struct.unpack_from("<I", buf, off)[0]
        struct.pack_into("<I", buf, off, replacement)
        print(f"  [OK] libil2cpp.so @ off 0x{off:x}: 0x{cur:08x} -> 0x{replacement:08x} ({desc})")
    return bytes(buf)


def replace_in_zip(src_zip: Path, dst_zip: Path, replacements: dict[str, bytes]):
    with zipfile.ZipFile(src_zip, "r") as zin, zipfile.ZipFile(dst_zip, "w") as zout:
        for item in zin.infolist():
            if item.filename in replacements:
                # libil2cpp.so precisa ficar STORED (sem compressão) para carregamento direto se extractNativeLibs=false
                if item.filename.endswith(".so"):
                    item.compress_type = zipfile.ZIP_STORED
                zout.writestr(item, replacements[item.filename])
            else:
                zout.writestr(item, zin.read(item.filename))


def main():
    parser = argparse.ArgumentParser(description="Gera os APKs modificados do Pokémon Champions para o Servidor Privado")
    parser.add_argument("--url", default="https://164.152.40.114:7443", help="URL do Servidor Privado (máx 35 chars)")
    parser.add_argument("--extracted-dir", default=str(DEFAULT_EXTRACTED_DIR), help="Pasta com base.apk e split_config.arm64_v8a.apk")
    parser.add_argument("--skip-so-patch", action="store_true", help="Não aplica patch em libil2cpp.so (usa AES-256-CBC nativo)")
    args = parser.parse_args()

    ext_dir = Path(args.extracted_dir)
    base_apk = ext_dir / "base.apk"
    arm64_apk = ext_dir / "split_config.arm64_v8a.apk"

    if not base_apk.exists():
        raise FileNotFoundError(f"base.apk não encontrado em {base_apk}")

    OUT_DIR.mkdir(parents=True, exist_ok=True)
    print(f"[1/3] Aplicando patch em global-metadata.dat e classes.dex para URL: {args.url}")
    with zipfile.ZipFile(base_apk, "r") as z:
        orig_md = z.read("assets/bin/Data/Managed/Metadata/global-metadata.dat")
        orig_dex = z.read("classes.dex")

    patched_md = patch_global_metadata(orig_md, args.url)
    patched_dex = patch_classes_dex(orig_dex)
    (OUT_DIR / "global-metadata.patched.dat").write_bytes(patched_md)

    out_base = OUT_DIR / "base.patched.apk"
    print(f"[2/3] Gerando {out_base.name} com global-metadata.dat e classes.dex modificados...")
    replace_in_zip(
        base_apk,
        out_base,
        {
            "assets/bin/Data/Managed/Metadata/global-metadata.dat": patched_md,
            "classes.dex": patched_dex,
        },
    )

    if not args.skip_so_patch and arm64_apk.exists():
        print("[3/3] Aplicando patch em libil2cpp.so dentro de split_config.arm64_v8a.apk...")
        with zipfile.ZipFile(arm64_apk, "r") as z:
            orig_so = z.read("lib/arm64-v8a/libil2cpp.so")
        patched_so = patch_libil2cpp(orig_so)
        out_arm64 = OUT_DIR / "split_config.arm64_v8a.patched.apk"
        replace_in_zip(arm64_apk, out_arm64, {"lib/arm64-v8a/libil2cpp.so": patched_so})
    else:
        print("[3/3] Pulando patch em libil2cpp.so (servidor já suporta criptografia nativa).")
        if arm64_apk.exists():
            shutil.copy2(arm64_apk, OUT_DIR / "split_config.arm64_v8a.patched.apk")

    print("\n" + "=" * 70)
    print(f"[SUCESSO] APKs modificados gerados em: {OUT_DIR}")
    print("Para assinar todos os splits (base + arm64 + UnityDataAssetPack) com a mesma chave:")
    print("  java -jar uber-apk-signer.jar -a build_apk/ --allowResign")
    print("=" * 70)


if __name__ == "__main__":
    main()
