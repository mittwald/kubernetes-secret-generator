FROM scratch
MAINTAINER Martin Helmich <m.helmich@mittwald.de>

COPY kubernetes-secret-generator /kubernetes-secret-generator

CMD ["/kubernetes-secret-generator", "-logtostderr"]