FROM debian:jessie

RUN apt-get update && apt-get -y upgrade && apt-get -y install curl wget git tar
RUN cd /tmp && wget https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz && tar -C /usr/local -xzf go1.8.linux-amd64.tar.gz 
ENV PATH=$PATH:/usr/local/go/bin

RUN cd /usr/src/ && git clone https://github.com/coreos/etcd.git -b release-2.3 && \
    cd /usr/src/etcd && \
    ./build && \
    ln -s /usr/src/etcd/bin/* /usr/bin/

ENV DISCOVER=shoutca.st

CMD etcd --proxy on --listen-client-urls http://127.0.0.1:2379 --discovery-srv $DISCOVER