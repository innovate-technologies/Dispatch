FROM debian:jessie
ENV container docker
ENV LC_ALL C
ENV DEBIAN_FRONTEND noninteractive
RUN cd /lib/systemd/system/sysinit.target.wants/; ls | grep -v systemd-tmpfiles-setup | xargs rm -f $1 \
rm -f /lib/systemd/system/multi-user.target.wants/*;\
rm -f /etc/systemd/system/*.wants/*;\
rm -f /lib/systemd/system/local-fs.target.wants/*; \
rm -f /lib/systemd/system/sockets.target.wants/*udev*; \
rm -f /lib/systemd/system/sockets.target.wants/*initctl*; \
rm -f /lib/systemd/system/basic.target.wants/*;\
rm -f /lib/systemd/system/anaconda.target.wants/*; \
rm -f /lib/systemd/system/plymouth*; \
rm -f /lib/systemd/system/systemd-update-utmp*;
RUN systemctl set-default multi-user.target
ENV init /lib/systemd/systemd

RUN apt-get update && apt-get -y upgrade && apt-get -y install curl wget git tar
RUN cd /tmp && wget https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz && tar -C /usr/local -xzf go1.8.linux-amd64.tar.gz 
ENV PATH=$PATH:/usr/local/go/bin

RUN cd /usr/src/ && git clone https://github.com/coreos/etcd.git -b release-2.3 && \
    cd /usr/src/etcd && \
    ./build && \
    ln -s /usr/src/etcd/bin/* /usr/bin/

ENV DISCOVER=shoutca.st


VOLUME [ "/sys/fs/cgroup" ]
ENTRYPOINT ["/lib/systemd/systemd"]