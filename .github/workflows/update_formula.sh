#!/bin/bash

# Replace these variables with your actual data
GITHUB_REPO="therealpaulgg/ssh-sync"  # Your GitHub username and repository name
FORMULA_PATH="Formula/ssh-sync.rb"  # Path to your formula in the tap
TAP_REPO="therealpaulgg/homebrew-ssh-sync"  # Your tap repository

# Fetch the latest release data from GitHub
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest")

# Extract the version and tarball URL from the release data
VERSION=$(echo "$LATEST_RELEASE" | jq -r '.tag_name')
TARBALL_URL=$(echo "$LATEST_RELEASE" | jq -r '.tarball_url')

# Download the tarball and calculate its SHA256
SHA256=$(curl -Ls $TARBALL_URL | shasum -a 256 | awk '{print $1}')

# Update the formula with the new version and sha256
sed -i "" "s|url \".*\"|url \"$TARBALL_URL\"|g" $FORMULA_PATH
sed -i "" "s|sha256 \".*\"|sha256 \"$SHA256\"|g" $FORMULA_PATH