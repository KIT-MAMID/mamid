#!/bin/bash
mkdir -p ca/private
mkdir -p ca/slaves
touch ca/index.txt

openssl req -new -keyout ./ca/private/mamid_private.pem -out ./ca/careq.pem -config ./scripts/openssl.cnf

openssl ca -config scripts/openssl.cnf -create_serial -out ./ca/mamid.pem -days 3650 -batch -keyfile ./ca/private/mamid_private.pem -selfsign -extensions v3_ca -infiles ./ca/careq.pem

chmod 400 ./ca/private/mamid_private.pem
echo "Generated CA in 'ca/mamid.pem' with its private key in './ca/private/mamid_private.pem'"
