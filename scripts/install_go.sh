#!/bin/bash

set -e

GO_VERSION="1.21.5"
GO_TAR="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_TAR}"
REMOTE_DIR="/home/yvivekan/lscc-blockchain"

echo "[+] Removing any existing Go installation..."
rm -rf /usr/local/go

echo "[+] Downloading Go ${GO_VERSION}..."
wget -q "${GO_URL}"

echo "[+] Extracting Go to /usr/local..."
tar -xzf "${GO_TAR}"

echo "[+] Updating ~/.bashrc with Go environment variables..."
{
  echo 'export PATH=$PATH:/home/yvivekan/go/bin'
  echo 'export GOPATH=$HOME/go'
  echo 'export GOBIN=$GOPATH/bin'
} >> ~/.bashrc

echo "[+] Applying changes to current shell..."
export PATH=$PATH:/home/yvivekan/go/bin
export GOPATH=$HOME/go
export GOBIN=$GOPATH/bin

echo "[+] Cleaning up..."
rm -f "${GO_TAR}"

cd ${REMOTE_DIR}
go mod tidy  
go build -o lscc-blockchain main.go

echo "[+] Completed Building lscc-blockchain"
