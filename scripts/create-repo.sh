#!/usr/bin/env bash
set -euo pipefail

# Build simple apt and rpm repositories from the package files in packages/
# Usage: create-repo.sh [repo-dir]

REPO_DIR=${1:-repo}
GPG_KEY_ID=${GPG_KEY_ID:-}

mkdir -p "$REPO_DIR/deb" "$REPO_DIR/rpm"

# Move packages into the repo
if compgen -G "packages/*.deb" > /dev/null; then
  mv packages/*.deb "$REPO_DIR/deb/"
fi
if compgen -G "packages/*.rpm" > /dev/null; then
  mv packages/*.rpm "$REPO_DIR/rpm/"
fi

if [ -n "$(ls -A "$REPO_DIR/deb" 2>/dev/null)" ]; then
  dpkg-scanpackages "$REPO_DIR/deb" /dev/null | gzip -9c > "$REPO_DIR/deb/Packages.gz"
  apt-ftparchive release "$REPO_DIR/deb" > "$REPO_DIR/deb/Release"
  if [ -n "$GPG_KEY_ID" ]; then
    gpg --batch --yes -u "$GPG_KEY_ID" -abs -o "$REPO_DIR/deb/Release.gpg" "$REPO_DIR/deb/Release"
  fi
fi

if [ -n "$(ls -A "$REPO_DIR/rpm" 2>/dev/null)" ]; then
  createrepo "$REPO_DIR/rpm"
  if [ -n "$GPG_KEY_ID" ]; then
    gpg --batch --yes -u "$GPG_KEY_ID" --detach-sign --armor -o "$REPO_DIR/rpm/repodata/repomd.xml.asc" "$REPO_DIR/rpm/repodata/repomd.xml"
  fi
fi
