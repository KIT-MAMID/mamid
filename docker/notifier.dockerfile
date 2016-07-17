FROM mongo:latest
WORKDIR /mamid

ADD build/notifier_linux_amd64 /mamid/notifier

CMD ["/mamid/notifier"]
