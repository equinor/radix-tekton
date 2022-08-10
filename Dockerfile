FROM golang:1.18.1-alpine3.15 as builder

ENV GO111MODULE=on

WORKDIR /go/src/github.com/equinor/radix-tekton/

# Install project dependencies
COPY go.mod go.sum ./

RUN apk update && apk upgrade
RUN apk add git ca-certificates curl && \
    apk add --no-cache gcc musl-dev && \
    go get -u golang.org/x/lint/golint

RUN go mod download

# Copy project code
COPY ./pkg ./pkg
COPY ./main.go ./

# Run tests
RUN go test -cover `go list ./... | grep -v 'pkg/client'`

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -a -installsuffix cgo -o ./rootfs/radix-tekton
RUN adduser -D -g '' -u 1000 radix-user

# Run
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /go/src/github.com/equinor/radix-tekton/rootfs/radix-tekton /usr/local/bin/radix-tekton
USER 1000
ENTRYPOINT ["/usr/local/bin/radix-tekton"]
