#!/bin/bash

echo "Updating system.."
sudo apt update -y
sudo apt upgrade -y

echo "Installing Git.."
sudo apt install git -y

echo "Installing Go.."
sudo apt install golang-go -y

echo "Creating app directory.."
mkdir -p ~/PublicDecisionTree

echo "Configuration complete."