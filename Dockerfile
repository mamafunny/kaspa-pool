FROM golang:1.19.1 as builder

ARG target=poolworker
ARG target_path=./cmd/${target}

WORKDIR /go/src/app
ADD go.mod .
ADD go.sum .
RUN go mod download

ADD cmd cmd
ADD src src
RUN go build -o /go/bin/app ./cmd/${target}

FROM gcr.io/distroless/base:nonroot
ARG target
COPY --from=builder /go/bin/app /
COPY cmd/${target}/config.yaml /

WORKDIR /
ENTRYPOINT ["/app"]
