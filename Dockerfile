# Build the manager binary
FROM golang:1.16 as builder

WORKDIR /workdir
# ENV GOPATH=/go
# Copy the Go Modules manifests
COPY go.mod go.sum /workdir/
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

RUN cat go.mod

# Copy the go source
COPY cmd cmd
COPY pkg pkg
COPY version version

RUN ls -la /workdir

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /workspace/manager ./cmd/manager/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
