#!/bin/bash
if [ ! -d ca ]; then
	echo "Plean run 'generateCA.sh' first."
	exit 1
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
openssl ca -config ./scripts/openssl.cnf -policy policy_anything -out ./ca/slaves/$1/$1.pem -infiles ./ca/slaves/$1/$1.csr
chmod 400 ./ca/slaves/${1}/${1}_key.pem
