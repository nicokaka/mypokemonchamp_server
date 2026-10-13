#!/usr/bin/env python3
"""
Script de Deploy Automático (1-Clique) para Oracle Cloud Free Tier
Servidor: MyPokeChampServer (Pokémon Champions v1.2.2-3859)
Coexiste pacificamente na porta 9443 ao lado do Pokémon TCG Pocket (porta 8443/443).
"""
import argparse
import os
import subprocess
import sys
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parent.parent
DEPLOY_DIR = ROOT_DIR / "DEPLOY_SERVIDOR"

DEFAULT_HOST = "164.152.40.114"
DEFAULT_USER = "ubuntu"
DEFAULT_KEY_CANDIDATES = [
    Path.home() / ".ssh" / "ssh-key-2026-09-19.key",
    Path.home() / "Downloads" / "ssh-key-2026-09-19.key",
    Path.home() / "Desktop" / "ssh-key-2026-09-19.key",
    ROOT_DIR.parent / "mytcgserver" / "ssh-key-2026-09-19.key",
]


def build_binaries():
    print("[1/3] Compilando binários Go para Linux (ARM64 Ampere A1 e AMD64)...")
    DEPLOY_DIR.mkdir(parents=True, exist_ok=True)
    for arch in ("arm64", "amd64"):
        out = DEPLOY_DIR / f"pokechamp-server-linux-{arch}"
        env = os.environ.copy()
        env["CGO_ENABLED"] = "0"
        env["GOOS"] = "linux"
        env["GOARCH"] = arch
        subprocess.run(
            ["go", "build", "-ldflags=-s -w", "-o", str(out), "./cmd/server"],
            cwd=str(ROOT_DIR),
            env=env,
            check=True,
        )
        print(f"  -> Gerado: {out.name} ({out.stat().st_size / (1024*1024):.2f} MB)")


def find_ssh_key(explicit_key: str | None) -> Path | None:
    if explicit_key:
        p = Path(explicit_key).expanduser().resolve()
        return p if p.exists() else None
    for cand in DEFAULT_KEY_CANDIDATES:
        if cand.exists():
            return cand
    return None


def main():
    parser = argparse.ArgumentParser(description="Deploy do MyPokeChampServer na Oracle Cloud")
    parser.add_argument("--host", default=DEFAULT_HOST, help="IP da VM Oracle Cloud")
    parser.add_argument("--user", default=DEFAULT_USER, help="Usuário SSH (padrão: ubuntu)")
    parser.add_argument("--key", default=None, help="Caminho para o arquivo .key SSH")
    parser.add_argument("--build-only", action="store_true", help="Apenas compila os binários Linux sem conectar via SSH")
    args = parser.parse_args()

    build_binaries()

    if args.build_only:
        print("\n[OK] Binários compilados em DEPLOY_SERVIDOR/ prontos para envio!")
        return

    ssh_key = find_ssh_key(args.key)
    if not ssh_key:
        print("\n" + "=" * 70)
        print("[AVISO] Chave SSH (.key) não encontrada neste computador.")
        print("Os binários Linux já foram compilados e estão prontos em:")
        print(f"  {DEPLOY_DIR}")
        print("\nQuando estiver no seu PC principal com a chave SSH, basta rodar:")
        print(f'  python scripts/deploy_champ_oracle.py --key "CAMINHO/PARA/ssh-key.key"')
        print("=" * 70)
        sys.exit(0)

    print(f"[2/3] Enviando pacote para {args.user}@{args.host} usando chave {ssh_key}...")
    ssh_opts = ["-i", str(ssh_key), "-o", "StrictHostKeyChecking=no"]

    subprocess.run(
        ["ssh", *ssh_opts, f"{args.user}@{args.host}", "mkdir -p /tmp/pokechamp_deploy"],
        check=True,
    )
    subprocess.run(
        [
            "scp",
            *ssh_opts,
            str(DEPLOY_DIR / "pokechamp-server-linux-arm64"),
            str(DEPLOY_DIR / "pokechamp-server-linux-amd64"),
            str(DEPLOY_DIR / "instalar_champ_oracle.sh"),
            f"{args.user}@{args.host}:/tmp/pokechamp_deploy/",
        ],
        check=True,
    )

    print("[3/3] Executando instalador systemd e liberando porta 9443 na Oracle Cloud...")
    remote_cmd = (
        "cd /tmp/pokechamp_deploy && "
        "sed -i 's/\r$//' instalar_champ_oracle.sh && "
        "chmod +x instalar_champ_oracle.sh && "
        "sudo ./instalar_champ_oracle.sh"
    )
    subprocess.run(["ssh", *ssh_opts, f"{args.user}@{args.host}", remote_cmd], check=True)


if __name__ == "__main__":
    main()
