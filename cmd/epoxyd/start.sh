#!/usr/bin/env bash

export PUBLIC_DIR="$(PWD)/public"

export PUBLIC_PREFIX=/

export ROUTES=$(cat <<EOF
    PrefixStrip /api/test http://localhost:29090
EOF
)

export BASIC_AUTH_PASS=test

go run -race main.go