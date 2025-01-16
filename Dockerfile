FROM --platform=$BUILDPLATFORM docker.io/golang:1.23.4-alpine3.21 AS builder
ARG TARGETARCH
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=${TARGETARCH}

WORKDIR /src

# Install project dependencies
COPY ./go.mod ./go.sum ./
RUN go mod download

COPY ./main.go ./
COPY ./pkg ./pkg
RUN go build -ldflags="-s -w" -o /build/radix-tekton

# Run
FROM gcr.io/distroless/static
WORKDIR /app
COPY --from=builder /build/radix-tekton .
USER 1000
ENTRYPOINT ["/app/radix-tekton"]
