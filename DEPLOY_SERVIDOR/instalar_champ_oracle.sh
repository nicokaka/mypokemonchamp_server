#!/usr/bin/env bash
set -e

if [ "$EUID" -ne 0 ]; then
    echo "[!] Execute como root: sudo ./instalar_champ_oracle.sh"
    exit 1
fi

INSTALL_DIR="/opt/pokechamp-server"
echo "=== Instalando Pokémon Champions Private Server 24/7 em $INSTALL_DIR ==="

mkdir -p "$INSTALL_DIR/bin" "$INSTALL_DIR/data" "$INSTALL_DIR/certs"

ARCH=$(uname -m)
if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    echo "[*] Arquitetura ARM64 (Oracle Ampere A1) detectada."
    cp -f ./pokechamp-server-linux-arm64 "$INSTALL_DIR/bin/pokechamp-server"
else
    echo "[*] Arquitetura x86_64 detectada."
    cp -f ./pokechamp-server-linux-amd64 "$INSTALL_DIR/bin/pokechamp-server"
fi

chmod 755 "$INSTALL_DIR/bin/pokechamp-server"

echo "[*] Liberando portas 9443 (HTTPS/WSS) e 9080 (Admin) no iptables da Oracle Cloud..."
if iptables -L INPUT -n --line-numbers | grep -q "REJECT"; then
    REJECT_LINE=$(iptables -L INPUT -n --line-numbers | grep "REJECT" | head -n1 | awk '{print $1}')
    iptables -C INPUT -p tcp --dport 9443 -j ACCEPT 2>/dev/null || iptables -I INPUT "$REJECT_LINE" -p tcp --dport 9443 -j ACCEPT
    iptables -C INPUT -p tcp --dport 9080 -j ACCEPT 2>/dev/null || iptables -I INPUT "$REJECT_LINE" -p tcp --dport 9080 -j ACCEPT
else
    iptables -C INPUT -p tcp --dport 9443 -j ACCEPT 2>/dev/null || iptables -A INPUT -p tcp --dport 9443 -j ACCEPT
    iptables -C INPUT -p tcp --dport 9080 -j ACCEPT 2>/dev/null || iptables -A INPUT -p tcp --dport 9080 -j ACCEPT
fi

if which netfilter-persistent >/dev/null 2>&1; then
    netfilter-persistent save 2>/dev/null || true
elif which iptables-save >/dev/null 2>&1; then
    mkdir -p /etc/iptables
    iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
fi

cat << 'SERVICE_EOF' > /etc/systemd/system/pokechamp.service
[Unit]
Description=Pokemon Champions 100% Private Server (MyPokeChampServer)
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/pokechamp-server
ExecStart=/opt/pokechamp-server/bin/pokechamp-server -addr :9443 -http-addr :9080 -public-host 164.152.40.114
Restart=always
RestartSec=3
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
SERVICE_EOF

systemctl daemon-reload
systemctl enable pokechamp.service
systemctl restart pokechamp.service

echo ""
echo "======================================================================"
echo "[SUCESSO] Servidor Pokémon Champions rodando na porta 9443 (HTTPS/WSS)!"
echo "Status: systemctl status pokechamp"
echo "Logs  : journalctl -u pokechamp -f"
echo "Admin : http://164.152.40.114:9080/admin"
echo "======================================================================"
