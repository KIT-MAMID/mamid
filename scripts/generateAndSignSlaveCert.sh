#!/bin/bash
if [ ! -d ca ]; then
	echo "Plean run 'generateCA.sh' first."
	exit 1
fi

echo '[INFO] Sign multiple certificates without repeated password prompts by using'
echo '       the CA_PASS environment variable:'
echo '         read -s -p "Enter CA signing key passphrase:: " CA_PASS'
echo '         export CA_PASS'
echo '         for i in $(seq -f '%02g' 1 99); do' $0 'slave${i}' '10.101.202.1${i}; done'

if [ -z "$*" ]; then
	echo "[ERR] No arguments specified"
	echo "      Usage: $0 commonName [IP address]"
	exit 1
fi

# prompt user for CA sign passphrase if not set
if [ -z "$CA_PASS"]; then
	read -s -p "Enter CA signing key passphrase: " CA_PASS
	export CA_PASS
else
	echo "[INFO] CA_PASS environment variable set, using contents as CA signing passphrase"
fi

mkdir -p ./ca/slaves/$1
CONF_STRING="[req]\nreq_extensions=ext\ndistinguished_name=dn\n[dn]\n[ext]\nsubjectAltName=DNS:$1"
if [ "$#" = 2 ]
then
	CONF_STRING="$CONF_STRING,IP:$2"
fi
echo $CONF_STRING
echo "---------------------------------------"
echo -e $CONF_STRING | openssl req -new -newkey rsa:4096 -keyout ./ca/slaves/${1}/${1}_key.pem -nodes -subj "/CN=$1" -config /dev/stdin -out ./ca/slaves/$1/$1.csr
openssl ca -batch -passin env:CA_PASS -config ./scripts/openssl.cnf -policy policy_anything -out ./ca/slaves/$1/$1.pem -infiles ./ca/slaves/$1/$1.csr
openssl pkcs12 -export -inkey ./ca/slaves/$1/$1_key.pem  -in ./ca/slaves/$1/$1.pem -passout pass: -out ./ca/slaves/$1/$1.p12
chmod 400 ./ca/slaves/${1}/${1}_key.pem

