#!/usr/bin/env bash
set -euo pipefail

# switch into the repo root directory
cd "$(dirname $0)"

PKGS=${PKGS:-$(go list ./... | xargs echo)}

echo -n "Running tests "
if [ ! -z "${COVERALLS:-""}" ]; then
    # coverage profile only works per-package
    echo "with coverage profile generation..."
    i=0
    for t in ${PKGS}; do
        go test -covermode set -coverprofile ${i}.coverprofile "${t}"
        i=$((i+1))
    done
else
    echo "without coverage profile generation..."
    go test ${PKGS}
fi

echo "Checking license header..."
licRes=$(
       for file in $(find . -type f -iname '*.go'); do
               head -n1 "${file}" | grep -Eq "(Copyright|generated)" || echo -e "  ${file}"
       done
)
if [ -n "${licRes}" ]; then
       echo -e "license header checking failed:\n${licRes}"
       exit 255
fi

echo "Success"
