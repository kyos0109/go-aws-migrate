FROM golang:1.14-stretch AS base

WORKDIR /go/src/app

COPY / .

RUN go get -d -v ./...

RUN go install -v ./...

FROM gcr.io/distroless/base

COPY --from=base /go/bin/go-migrate-security-group /usr/bin/go-migrate-security-group

WORKDIR /app

ENTRYPOINT ["go-migrate-security-group"]