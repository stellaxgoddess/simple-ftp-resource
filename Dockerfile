FROM golang AS builder

ENV PACKAGE=github.com/pawel20987/simple-ftp-resource

RUN mkdir -p /go/src/${PACKAGE}
WORKDIR /go/src/${PACKAGE}

ENV LD_FLAGS="-w"
ENV CGO_ENABLED=0

COPY . /go/src/${PACKAGE}
RUN go install -a -v -tags netgo -ldflags "${LD_FLAGS}" .

FROM busybox

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/bin/simple-ftp-resource /opt/resource/simple-ftp-resource

WORKDIR /opt/resource/

RUN ln -s simple-ftp-resource check \
 && ln -s simple-ftp-resource in \
 && ln -s simple-ftp-resource out
