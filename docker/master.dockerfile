FROM mongo:latest
WORKDIR /mamid

ADD build/master_linux_amd64 /mamid/master

CMD ["/mamid/master"]
