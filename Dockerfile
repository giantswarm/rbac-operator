FROM gsoci.azurecr.io/giantswarm/alpine:3.23.3

RUN apk add --no-cache ca-certificates

ADD ./rbac-operator /rbac-operator

ENTRYPOINT ["/rbac-operator"]
