FROM mongo:latest
WORKDIR /mamid

ADD build/master_docker /mamid/master

CMD ["/mamid/master", "-slave.verifyCA=/mamid/ca.pem", "-slave.auth.key=/mamid/master_key.pem", "-slave.auth.cert=/mamid/master.pem"]
