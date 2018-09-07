FROM golang:1.11

COPY . /go/src/github.com/mittwald/kubernetes-secret-generator
WORKDIR /go/src/github.com/mittwald/kubernetes-secret-generator
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kubernetes-secret-generator

FROM scratch
MAINTAINER Martin Helmich <m.helmich@mittwald.de>

COPY --from=0 /go/src/github.com/mittwald/kubernetes-secret-generator/kubernetes-secret-generator /kubernetes-secret-generator

CMD ["/kubernetes-secret-generator", "-logtostderr"]
