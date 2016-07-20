FROM mongo:latest
WORKDIR /mamid

ADD build/slave_docker /mamid/slave

CMD ["/mamid/slave"]
