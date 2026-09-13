#!/usr/bin/env bash
set -euo pipefail

JCLI_HOST="${JCLI_HOST:-127.0.0.1}"
JCLI_PORT="${JCLI_PORT:-8990}"
JCLI_ADMIN_USERNAME="${JCLI_ADMIN_USERNAME:-jcliadmin}"
JCLI_ADMIN_PASSWORD="${JCLI_ADMIN_PASSWORD:-jclipwd}"
JASMIN_GROUP_ID="${JASMIN_GROUP_ID:-callio}"
JASMIN_USERNAME="${JASMIN_USERNAME:-callio_user}"
JASMIN_PASSWORD="${JASMIN_PASSWORD:-callio_secret}"

python3 - "$JCLI_HOST" "$JCLI_PORT" "$JCLI_ADMIN_USERNAME" "$JCLI_ADMIN_PASSWORD" "$JASMIN_GROUP_ID" "$JASMIN_USERNAME" "$JASMIN_PASSWORD" <<'PY'
import socket
import sys
import time

host, port, admin_user, admin_pass, group_id, username, password = sys.argv[1:]

def read_until(sock, needles, timeout=15):
    if isinstance(needles, str):
        needles = [needles]
    deadline = time.time() + timeout
    data = b""
    while time.time() < deadline:
        try:
            chunk = sock.recv(4096)
        except socket.timeout:
            continue
        if not chunk:
            break
        data += chunk
        text = data.decode(errors="replace")
        if any(needle in text for needle in needles):
            return text
    raise TimeoutError(f"Timed out waiting for {needles!r}. Last output:\n{data.decode(errors='replace')}")

def send_line(sock, line):
    sock.sendall((line + "\n").encode())

with socket.create_connection((host, int(port)), timeout=10) as sock:
    sock.settimeout(1)
    first = read_until(sock, ["Username:", "Authentication required."])
    send_line(sock, admin_user)
    read_until(sock, "Password:")
    send_line(sock, admin_pass)
    read_until(sock, "jcli :")

    commands = [
        "group -a",
        f"gid {group_id}",
        "ok",
        "user -a",
        f"username {username}",
        f"password {password}",
        f"gid {group_id}",
        f"uid {username}",
        "ok",
        "persist",
    ]

    transcript = first
    for command in commands:
        send_line(sock, command)
        transcript += read_until(sock, "jcli :" if command in {"ok", "persist"} else "> ")

print(f"Requested Jasmin group/user setup for {username!r} in group {group_id!r}.")
print("If the group or user already existed, Jasmin may report that in its console output; matching .env credentials are still shown below.")
print(f"JASMIN_USERNAME={username}")
print(f"JASMIN_PASSWORD={password}")
PY
