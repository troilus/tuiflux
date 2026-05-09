import json
import os
from pathlib import Path

CONFIG_FILE = Path("config.json")

def load_config():
    if not CONFIG_FILE.exists():
        return setup_config()
    
    try:
        with open(CONFIG_FILE, "r", encoding="utf-8") as f:
            return json.load(f)
    except (json.JSONDecodeError, IOError):
        return setup_config()

def setup_config():
    print("--- Miniflux TUI Setup ---")
    server_url = input("Server URL (e.g., https://miniflux.example.com): ").strip()
    api_key = input("API Key: ").strip()
    verify_ssl_input = input("Verify SSL certificates? (Y/n): ").strip().lower()
    verify_ssl = verify_ssl_input != 'n'
    
    config = {
        "server_url": server_url.rstrip("/"),
        "api_key": api_key,
        "verify_ssl": verify_ssl
    }
    
    with open(CONFIG_FILE, "w", encoding="utf-8") as f:
        json.dump(config, f, indent=4)
    
    print(f"Configuration saved to {CONFIG_FILE}")
    return config
