FROM --platform=$BUILDPLATFORM golang:1.25.7 AS builder

WORKDIR /app

COPY go-binary .

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o kubara main.go


FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/kubara /kubara

ENTRYPOINT ["/kubara"]