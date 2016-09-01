FROM mongo:latest
WORKDIR /mamid

ADD build/slave_docker /mamid/slave
RUN mkdir /slave

CMD ["/mamid/slave", "-data=/slave", "-serverCertFile=/mamid/cert.pem", "-serverKeyFile=/mamid/key.pem", "-cacert=/mamid/ca.pem"]
