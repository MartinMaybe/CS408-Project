#!/bin/bash
set -e

APP_DIR=~/PublicDecisionTree

echo "Entering app directory.."
cd $APP_DIR

if [ -d ".git" ]; then
    echo "Pulling latest changes.."
    git pull origin main
else 
    echo "Cloning repository.."
    git clone https://github.com/MartinMaybe/CS408-Project .
fi

echo "Downloading dependencies.."
go mod tidy

echo "Building Go application.."
go build -o server

echo "Stopping existing server.."
pkill server || true

echo "Starting server.."
nohup ./server > app.log 2>&1 &