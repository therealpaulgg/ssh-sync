#!/usr/bin/env bash
set -euo pipefail

# Build simple apt and rpm repositories from the package files in packages/

REPO_DIR=${1:-repo}

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
fi

if [ -n "$(ls -A "$REPO_DIR/rpm" 2>/dev/null)" ]; then
  createrepo "$REPO_DIR/rpm"
fi
