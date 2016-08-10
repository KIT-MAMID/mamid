FROM mongo:latest
WORKDIR /mamid

ADD build/slave_docker /mamid/slave
RUN mkdir /slave

CMD ["/mamid/slave", "-data=/slave"]
