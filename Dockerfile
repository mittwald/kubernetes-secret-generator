FROM        golang:1.12
WORKDIR     /src
COPY        . .
RUN         CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kubernetes-secret-generator

FROM        scratch
LABEL       MAINTAINER="Martin Helmich <m.helmich@mittwald.de>"
COPY        --from=0 /src/kubernetes-secret-generator /kubernetes-secret-generator
ENTRYPOINT  ["/kubernetes-secret-generator"]
CMD         ["-logtostderr"]