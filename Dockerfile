FROM debian:jessie

RUN echo "deb http://ftp.debian.org/debian jessie-backports main" >>/etc/apt/sources.list
RUN apt-get update && apt-get -y upgrade && apt-get -y install curl git
RUN apt-get -y install -t jessie-backports golang

RUN cd /usr/src/ && git clone https://github.com/coreos/etcd.git -b release-2.3 && \
    cd /usr/src/etcd && \
    ./build && \
    ln -s /usr/src/etcd/bin/* /usr/bin/

ENV DISCOVER=shoutca.st
ENV PUBLICIP=127.0.0.1
ENV MACHINENAME=dev
ENV TAGS=zone=par1,model=C1

COPY ./dispatch /opt/dispatch
COPY ./dispatchctl /opt/dispatchctl
COPY ./dispatchctl/dispatchctl /usr/local/bin/dispatchctl
RUN chmod +x /usr/local/bin/dispatchctl

CMD etcd --proxy on --listen-client-urls http://127.0.0.1:2379 --discovery-srv $DISCOVER