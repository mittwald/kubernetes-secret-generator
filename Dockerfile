FROM        scratch
LABEL       MAINTAINER="Martin Helmich <m.helmich@mittwald.de>"
COPY        kubernetes-secret-generator /kubernetes-secret-generator
ENTRYPOINT  ["/kubernetes-secret-generator"]
CMD         ["-logtostderr"]