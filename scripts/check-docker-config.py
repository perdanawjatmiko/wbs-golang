#!/usr/bin/env python3
"""Static Compose policy checks only. Never starts services or connects to a DB."""
import json
import os
from pathlib import Path
import subprocess

root = Path(__file__).resolve().parents[1]
env = dict(os.environ, WBS_ENV_FILE=str(root / ".env.docker.example"))
result = subprocess.run(
    ["docker", "compose", "--env-file", "/dev/null", "-f", str(root / "docker-compose.yml"),
     "config", "--no-env-resolution", "--format", "json"],
    cwd=root, env=env, capture_output=True, text=True, check=True, timeout=30,
)
config = json.loads(result.stdout)
assert set(config["services"]) == {"frontend", "backend", "worker"}
assert set(config.get("volumes", {})) == {"attachments"}
for name, service in config["services"].items():
    assert service["read_only"] and service["cap_drop"] == ["ALL"]
    assert "no-new-privileges:true" in service["security_opt"]
    assert not service.get("privileged") and service.get("network_mode") != "host"
    assert service["user"] not in ("root", "0", "0:0")
    for port in service.get("ports", []):
        assert port["host_ip"] == "127.0.0.1"
        assert port["target"] in (3000, 8080)
    assert int(service["pids_limit"]) <= 256 and int(service["mem_limit"]) <= 1024 ** 3
worker = config["services"]["worker"]
assert not worker.get("ports") and not worker.get("volumes")
assert config["services"]["backend"]["image"] == worker["image"]
assert config["services"]["backend"]["environment"]["SMTP_ENABLED"] == "false"
assert config["services"]["backend"]["environment"]["TELEGRAM_ENABLED"] == "false"
assert config["services"]["frontend"]["environment"]["API_BASE_URL"] == "http://backend:8080"
assert "DATABASE_URL" not in config["services"]["frontend"]["environment"]
for service in config["services"].values():
    for mount in service.get("volumes", []):
        assert mount["type"] == "volume" and mount["source"] == "attachments"
print("PASS: Compose structure/security policy only; no runtime or database verification")
