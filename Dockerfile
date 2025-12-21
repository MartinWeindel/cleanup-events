############# builder
FROM golang:1.25.5 AS builder

WORKDIR /src

# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

COPY . .

RUN make install

############# cleanup-events

FROM gcr.io/distroless/static-debian13:nonroot AS cleanup-events
WORKDIR /

LABEL org.opencontainers.image.source=„

COPY --from=builder /go/bin/cleanup-events /cleanup-events
ENTRYPOINT ["/cleanup-events"]
