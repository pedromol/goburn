FROM golang:alpine as build

WORKDIR /go/src/app

COPY main.go /go/src/app
COPY go.mod /go/src/app

RUN CGO_ENABLED=0 go build -ldflags '-extldflags "-static" -w -s' -tags timetzdata

FROM scratch

COPY --from=build /go/src/app/goburn /main

ENTRYPOINT ["/main"]
