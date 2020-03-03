FROM ubuntu:19.10
MAINTAINER Raphael Reyna <raphaelreyna@protonmail.com>
WORKDIR /latte
RUN apt update -q \
	&& env DEBIAN_FRONTEND=noninteractive \
	apt install -qy texlive-full golang-1.13 \
	&& rm -rf /var/lib/apt/lists/*
COPY ./ /latte
RUN /usr/lib/go-1.13/bin/go build .
CMD ["/latte/latte"]

