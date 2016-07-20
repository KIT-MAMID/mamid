FROM mongo:latest
WORKDIR /

RUN apt-get update
RUN apt-get install -y build-essential curl

# install go into /go directory
RUN curl https://storage.googleapis.com/golang/go1.6.2.linux-amd64.tar.gz | tar xfz -

ENV GOROOT=/go
ENV PATH=$GOROOT/bin:$PATH
ENV GOPATH=/gopath

RUN go env





