#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." &> /dev/null && pwd )"

pushd "${DIR}" &> /dev/null

./scripts/build.sh
npm -v &> /dev/null
release-it

popd &> /dev/null