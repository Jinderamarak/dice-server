# Build step
FROM --platform=${BUILDPLATFORM} golang:1 AS builder
WORKDIR /build

ARG TARGETOS
ARG TARGETARCH

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o app main.go

# Runtime step
FROM scratch
WORKDIR /app

COPY --from=builder /build/app /app/dice-server

EXPOSE 9000
ENV GIN_MODE=release
ENTRYPOINT ["/app/dice-server"]
