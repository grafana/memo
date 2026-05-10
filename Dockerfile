FROM golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

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