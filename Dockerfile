FROM debian:jessie

RUN echo "deb http://ftp.debian.org/debian jessie-backports main" >>/etc/apt/sources.list
RUN apt-get update && apt-get -y upgrade && apt-get -y install curl git
RUN apt-get -y install -t jessie-backports golang

RUN cd /usr/src/ && git clone https://github.com/coreos/etcd.git -b release-2.3 && \
    cd /usr/src/etcd && \
    ./build && \
    ln -s /usr/src/etcd/bin/* /usr/bin/

ENV GOPATH=/tmp/go
ENV DISCOVER=shoutca.st

CMD etcd --proxy on --listen-client-urls http://127.0.0.1:2379 --discovery-srv $DISCOVER