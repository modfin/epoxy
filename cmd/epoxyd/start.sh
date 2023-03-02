#!/usr/bin/env bash

export PUBLIC_DIR="$(PWD)/public"

export PUBLIC_PREFIX=/

export ROUTES=$(cat <<EOF
    PrefixStrip /api/test http://localhost:29090
EOF
)

export BASIC_AUTH_PASS=test

export EPOXY_JWT_EC_256=$(openssl ecparam -name prime256v1 -genkey -noout)

go run -race main.go