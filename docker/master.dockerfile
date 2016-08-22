FROM mongo:latest
WORKDIR /mamid

ADD build/master_docker /mamid/master

CMD ["/mamid/master", "-db.path","/mamid/mamid.sqlite3"]
