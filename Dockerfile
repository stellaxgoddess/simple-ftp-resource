FROM golang:1 AS builder


RUN mkdir -p /workdir
WORKDIR /workdir

COPY . /workdir
RUN go mod download && go build

FROM busybox

COPY --from=builder /workdir/simple-ftp-resource /opt/resource/simple-ftp-resource

WORKDIR /opt/resource/

RUN ln -s simple-ftp-resource check \
 && ln -s simple-ftp-resource in \
 && ln -s simple-ftp-resource out
