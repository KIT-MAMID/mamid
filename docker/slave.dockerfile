FROM mongo:latest
WORKDIR /mamid

ADD build/slave_linux_amd64 /mamid/slave

CMD ["/mamid/slave"]
