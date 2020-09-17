FROM golang:1.14-stretch AS base

WORKDIR /go/src/app

COPY / .

RUN go get -d -v ./...

RUN go install -v ./...

FROM gcr.io/distroless/base

COPY --from=base /go/bin/go-aws-migrate /usr/bin/go-aws-migrate

WORKDIR /app

ENTRYPOINT ["go-aws-migrate"]