FROM centos/systemd

COPY dispatchd.service /etc/systemd/system/dispatchd.service
COPY ./dispatchd /usr/bin/dispatchd
COPY ./dispatchctl /usr/bin/dispatchctl
RUN systemctl enable dispatchd.service
