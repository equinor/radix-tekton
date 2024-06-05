FROM golang:1.22-alpine3.20 as builder

RUN apk update && apk add git ca-certificates curl && \
    apk add --no-cache gcc musl-dev

WORKDIR /go/src/github.com/equinor/radix-tekton/

# Install project dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy project code
COPY ./pkg ./pkg
COPY ./main.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -a -installsuffix cgo -o /radix-tekton
RUN adduser -D -g '' -u 1000 radix-user

# Run
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /radix-tekton /radix-tekton
USER 1000
ENTRYPOINT ["/radix-tekton"]
