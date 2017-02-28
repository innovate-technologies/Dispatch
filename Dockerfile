FROM golang:1.8

RUN apt-get update && apt-get -y upgrade && apt-get -y install git


RUN cd /usr/src/ && git clone https://github.com/coreos/etcd.git -b release-2.3 && \
    cd /usr/src/etcd && \
    ./build && \
    ln -s /usr/src/etcd/bin/* /usr/bin/

ENV DISCOVER=shoutca.st

COPY ./dispatchd /usr/src/dispatchd

RUN cd /usr/src/dispatchd && go get -v -d && go build dispatchd.go && mv dispatchd /usr/bin/

CMD etcd --proxy on --listen-client-urls http://127.0.0.1:2379 --discovery-srv $DISCOVER