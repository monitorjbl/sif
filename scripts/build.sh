#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." &> /dev/null && pwd )"

pushd "${DIR}" &> /dev/null

rm -fr ./dist
mkdir ./dist

echo "Building darwin-x64"
GOOS=darwin GOARCH=amd64 go build -o ./dist/sif-darwin-x64
echo "Building linux-x64"
GOOS=linux GOARCH=amd64 go build -o ./dist/sif-linux-x64
echo "Building windows-x64"
GOOS=windows GOARCH=amd64 go build -o ./dist/sif-windows-x64.exe

popd &> /dev/null