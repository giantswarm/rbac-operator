FROM gsoci.azurecr.io/giantswarm/alpine:3.24.3

RUN apk add --no-cache ca-certificates

ARG TARGETARCH
COPY rbac-operator-linux-${TARGETARCH} /rbac-operator

ENTRYPOINT ["/rbac-operator"]
