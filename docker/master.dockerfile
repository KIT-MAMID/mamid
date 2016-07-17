FROM mongo:latest
WORKDIR /mamid

ADD build/master_linux_amd64 /mamid/master
ADD gui /mamid/gui

CMD ["/mamid/master"]
