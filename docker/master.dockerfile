FROM mongo:latest
WORKDIR /mamid

ADD build/master_docker /mamid/master

CMD ["/mamid/master", "-cacert=/mamid/ca.pem", "-clientKey=/mamid/master_key.pem", "-clientCert=/mamid/master.pem"]
