FROM 1.18.0-alpine3.15 as builder

ENV GO111MODULE=on

RUN apk update && \
    apk add git ca-certificates curl && \
    apk add --no-cache gcc musl-dev && \
    go get -u golang.org/x/lint/golint github.com/frapposelli/wwhrd@v0.2.4

WORKDIR /go/src/github.com/equinor/radix-tekton/

# Install project dependencies
COPY go.mod go.sum ./
RUN go mod download

# Check dependency licenses using https://github.com/frapposelli/wwhrd
COPY .wwhrd.yml ./
RUN wwhrd -q check

# Copy project code
COPY ./cmd ./cmd
COPY ./pkg ./pkg

WORKDIR /go/src/github.com/equinor/radix-tekton/cmd

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -a -installsuffix cgo -o ./rootfs/radix-tekton
RUN adduser -D -g '' -u 1000 radix-user

# Run
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /go/src/github.com/equinor/radix-tekton/cmd/rootfs/radix-config-2-map /usr/local/bin/radix-tekton
USER 1000
ENTRYPOINT ["/usr/local/bin/radix-tekton"]
