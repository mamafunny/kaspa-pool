FROM golang:1.19.1 as builder

WORKDIR /go/src/app
ADD go.mod .
ADD go.sum .
RUN go mod download

ADD cmd cmd
ADD src src
RUN go build -o /go/bin/app ./cmd/poolworker

FROM gcr.io/distroless/base:nonroot
COPY --from=builder /go/bin/app /
COPY cmd/poolworker/config.yaml /

WORKDIR /
ENTRYPOINT ["/app"]
