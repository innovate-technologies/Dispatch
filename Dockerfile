FROM debian:jessie

RUN apt-get update && apt-get -y upgrade && apt-get -y install curl
RUN curl -sL https://deb.nodesource.com/setup_6.x | bash - && apt-get install nodejs

ENV DISCOVER=shoutca.st
ENV PUBLICIP=127.0.0.1
ENV MACHINENAME=dev
ENV TAGS=zone=par1,model=C1
