ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

# Build step
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1 as builder
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o app main.go

# Runtime step
FROM --platform=${TARGETPLATFORM:-linux/amd64} scratch
WORKDIR /app

COPY --from=builder /build/app /app/dice-server

EXPOSE 9000
ENV GIN_MODE=release
CMD ["./dice-server"]
