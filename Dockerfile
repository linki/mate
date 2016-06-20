FROM golang:1.6.2-alpine

RUN apk --no-cache add ca-certificates gcc g++

RUN mkdir -p /go/src/app
WORKDIR /go/src/app

ENTRYPOINT ["/go/bin/app"]
CMD []

COPY vendor /go/src/app/vendor
COPY main.go /go/src/app/

RUN go get -v -d
RUN go install -v
