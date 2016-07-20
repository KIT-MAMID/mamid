FROM mongo:latest
WORKDIR /mamid

ADD build/notifier_docker /mamid/notifier

CMD ["/mamid/notifier"]
