FROM golang:1.27-alpine@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS builder

ADD . /opt/memo

WORKDIR /opt/memo

RUN apk --update add --no-cache ca-certificates openssl git tzdata && \
  update-ca-certificates

RUN go get -v && \
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o memod cmd/memod/* && \
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o memo-cli cmd/memo-cli/* && \
  chmod +x memod memo-cli

FROM alpine:latest

COPY --from=builder /opt/memo/memod /bin/memod
COPY --from=builder /opt/memo/memo-cli /bin/memo-cli

CMD [ "memod" ]