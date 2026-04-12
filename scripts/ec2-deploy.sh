#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

prompt_for_git_pull() {
    if [ ! -d ".git" ]; then
        echo "No .git directory found; skipping git pull."
        return
    fi

    if [ -t 0 ]; then
        read -r -p "Pull latest changes from origin/main? [Y/n] " pull_choice
        case "${pull_choice:-Y}" in
            [Yy]|[Yy][Ee][Ss])
                echo "Pulling latest changes.."
                git pull --ff-only origin main
                ;;
            [Nn]|[Nn][Oo])
                echo "Skipping git pull."
                ;;
            *)
                echo "Unrecognized choice; skipping git pull."
                ;;
        esac
        return
    fi

    echo "Non-interactive shell detected; pulling latest changes by default."
    git pull --ff-only origin main
}

echo "Entering app directory: $APP_DIR"
cd "$APP_DIR"

if [ ! -f "go.mod" ]; then
    echo "go.mod not found in $APP_DIR"
    echo "Run this script from a checked-out copy of the repository."
    exit 1
fi

prompt_for_git_pull

echo "Downloading dependencies.."
go mod tidy

echo "Building Go application.."
go build -o server .

echo "Stopping existing server.."
pkill -f "$APP_DIR/server" || true

echo "Starting server.."
nohup "$APP_DIR/server" > "$APP_DIR/app.log" 2>&1 &

echo "Deployment complete."
