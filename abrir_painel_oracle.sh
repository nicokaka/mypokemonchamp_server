#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KEY_CANDIDATES=(
  "$DIR/../tcg-makemoney/ssh-key-2026-09-19.key"
  "$HOME/.ssh/ssh-key-2026-09-19.key"
  "$HOME/Downloads/ssh-key-2026-09-19.key"
)
KEY=""
for k in "${KEY_CANDIDATES[@]}"; do
  if [ -f "$k" ]; then
    KEY="$k"
    break
  fi
done

ORACLE_IP="164.152.40.114"
LOCAL_PORT=19080

echo "=== 🏆 MyPokeChamp Studio — Painel de Controle Remoto (Oracle Cloud 24/7) ==="

if [ -n "$KEY" ]; then
  if ! pgrep -f "$LOCAL_PORT:127.0.0.1:9080" > /dev/null; then
    echo "[*] Conectando túnel SSH seguro com o Oracle Cloud ($ORACLE_IP)..."
    ssh -i "$KEY" -o StrictHostKeyChecking=no -f -N -L "$LOCAL_PORT:127.0.0.1:9080" "ubuntu@$ORACLE_IP"
    sleep 1
  fi
  echo "[✓] Conexão segura estabelecida!"
  echo "👉 Acesse no navegador (PC): http://localhost:$LOCAL_PORT/admin"
  echo "👉 Ou direto via HTTPS (PC/Celular): https://$ORACLE_IP:9443/admin"
  xdg-open "http://localhost:$LOCAL_PORT/admin" 2>/dev/null || true
else
  echo "👉 Acesse direto no navegador: https://$ORACLE_IP:9443/admin"
  xdg-open "https://$ORACLE_IP:9443/admin" 2>/dev/null || true
fi
