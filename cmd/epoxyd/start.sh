#!/usr/bin/env bash

export PUBLIC_DIR="$(PWD)/public"
export PUBLIC_PREFIX=/

export ROUTES=$(cat <<EOF
    PrefixStrip /api/test http://localhost:29090
EOF
)

export DEV_PASS=test
export DEV_BCRYPT_HASH=$(htpasswd -bnBC 10 "" "$DEV_PASS" | tr -d ':\n')
export DEV_SESSION_DURATION=1m
export JWT_EC_256=$(openssl ecparam -name prime256v1 -genkey -noout)
export JWT_EC_256_PUB=$(echo "$JWT_EC_256" | openssl ec -pubout)

go run -race main.go