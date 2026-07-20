FROM golang:1.19-alpine@sha256:0ec0646e208ea58e5d29e558e39f2e59fccf39b7bda306cb53bbaff91919eca5 AS builder

ADD . /opt/memo

WORKDIR /opt/memo

RUN apk --update add --no-cache ca-certificates openssl git tzdata && \
  update-ca-certificates

RUN go get -v && \
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o memod cmd/memod/* && \
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o memo-cli cmd/memo-cli/* && \
  chmod +x memod memo-cli

FROM alpine:latest@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

COPY --from=builder /opt/memo/memod /bin/memod
COPY --from=builder /opt/memo/memo-cli /bin/memo-cli

CMD [ "memod" ]