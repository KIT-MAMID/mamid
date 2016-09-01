FROM mongo:latest
WORKDIR /mamid

ADD build/slave_docker /mamid/slave
ADD docker/keyfile /mamid/keyfile
RUN chmod 0400 /mamid/keyfile
RUN mkdir /slave

CMD ["/mamid/slave", "-data=/slave", "-slave.auth.cert=/mamid/cert.pem", "-slave.auth.key=/mamid/key.pem", "-master.verifyCA=/mamid/ca.pem"]
