FROM docker.io/golang:1.22.5-alpine3.20 AS builder

ENV CGO_ENABLED=0 \
     GOOS=linux

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
