FROM centos/systemd

RUN curl https://get.docker.com | bash

COPY dispatchd.service /etc/systemd/system/dispatchd.service
COPY ./dispatchd /usr/bin/dispatchd
COPY ./dispatchctl /usr/bin/dispatchctl

RUN systemctl enable docker.service
RUN systemctl enable dispatchd.service

COPY ./test.service /test.service