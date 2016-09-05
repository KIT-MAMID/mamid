FROM mongo:latest
WORKDIR /mamid

RUN apt-get update
RUN apt-get install -y ca-certificates

ADD build/notifier_docker /mamid/notifier

CMD ["/mamid/notifier"]
