#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

echo "=== 🏆 Instalador Pokémon Champions (v1.2.2-3859) para Celular Android ==="

echo "[1/4] Procurando celular Android conectado via USB (com Depuração USB ativa)..."
DEVICES=$(adb devices -l | grep "device " | grep -v "192.168.240" | awk '{print $1}')

if [ -z "$DEVICES" ]; then
    echo "[!] Nenhum celular físico detectado via ADB!"
    exit 1
fi

SERIAL=$(echo "$DEVICES" | head -n1)
MODEL=$(adb -s "$SERIAL" shell getprop ro.product.model 2>/dev/null | tr -d '\r')
ANDROID_VER=$(adb -s "$SERIAL" shell getprop ro.build.version.release 2>/dev/null | tr -d '\r')
echo "  -> Celular detectado: $MODEL (Serial: $SERIAL, Android $ANDROID_VER)"

ZIPALIGN="/home/nicokaka/android-sdk/build-tools/35.0.0/zipalign"
APKSIGNER="/home/nicokaka/android-sdk/build-tools/35.0.0/apksigner"
KEYSTORE="/home/nicokaka/Documents/projects/tcg-makemoney/MODDED_APK/debug.keystore"

mkdir -p "$DIR/build_apk"

if [ ! -f "$DIR/build_apk/base.patched.apk" ] || [ ! -f "$DIR/build_apk/split_config.arm64_v8a.patched.apk" ] || [ "$DIR/scripts/gerar_apk_modificado.py" -nt "$DIR/build_apk/base.patched.apk" ]; then
    echo "[2/4] Gerando APKs modificados para o servidor Oracle Cloud (https://164.152.40.114:7443)..."
    python3 "$DIR/scripts/gerar_apk_modificado.py"
fi

sign_one() {
    local src="$1"
    local name="$2"
    local aligned="$DIR/build_apk/aligned_${name}"
    local signed="$DIR/build_apk/signed_${name}"

    if [ ! -f "$signed" ] || [ "$src" -nt "$signed" ]; then
        echo "  -> Alinhando e assinando $name..."
        "$ZIPALIGN" -f -p 4 "$src" "$aligned"
        "$APKSIGNER" sign \
            --ks "$KEYSTORE" \
            --ks-pass pass:android \
            --key-pass pass:android \
            --ks-key-alias androiddebugkey \
            --out "$signed" \
            "$aligned"
        rm -f "$aligned"
    fi
}

echo "[3/4] Verificando assinatura unificada dos 3 Split APKs..."
sign_one "$DIR/build_apk/base.patched.apk" "base.apk"
sign_one "$DIR/build_apk/split_config.arm64_v8a.patched.apk" "split_config.arm64_v8a.apk"
sign_one "$DIR/apk/extracted/split_UnityDataAssetPack.apk" "split_UnityDataAssetPack.apk"

echo "[4/4] Instalando Pokémon Champions no celular ($MODEL)..."
adb -s "$SERIAL" install-multiple -r -i com.android.vending --user 0 --no-incremental \
    "$DIR/build_apk/signed_base.apk" \
    "$DIR/build_apk/signed_split_config.arm64_v8a.apk" \
    "$DIR/build_apk/signed_split_UnityDataAssetPack.apk"

echo "[*] Iniciando Pokémon Champions no aparelho..."
adb -s "$SERIAL" shell am start --user 0 -n jp.pokemon.pokemonchampions/com.unity3d.player.UnityPlayerActivity >/dev/null 2>&1 || true

echo ""
echo "========================================================================="
echo "[SUCESSO] Pokémon Champions instalado no seu celular ($MODEL)!"
echo "Servidor Privado: https://164.152.40.114:7443"
echo "Painel de Perfil: http://localhost:19080/admin (ou http://164.152.40.114:9080/admin)"
echo "========================================================================="
