# Build step
FROM --platform=${BUILDPLATFORM} golang:1 AS builder
WORKDIR /build

ARG TARGETOS
ARG TARGETARCH

COPY go.work go.work.sum ./
COPY common/go.mod ./common/
COPY game/common/go.mod ./game/common/
COPY game/farkle/go.mod ./game/farkle/
COPY game-master/go.mod ./game-master/
RUN go mod download

COPY .. .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o app game/farkle/main.go

# Runtime step
FROM scratch
WORKDIR /app

COPY --from=builder /build/app /app/game-farkle

ENV GIN_MODE=release
ENTRYPOINT ["/app/game-farkle"]
