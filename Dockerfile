FROM --platform=$BUILDPLATFORM golang:1.25.7 AS builder

WORKDIR /app

COPY go-binary .

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o kubara main.go


FROM alpine

COPY --from=builder /app/kubara /kubara

ENTRYPOINT ["/kubara", "--kubeconfig", "/kubeconfig"]