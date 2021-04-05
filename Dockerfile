FROM alpine:3.13.4

RUN apk add --no-cache ca-certificates

ADD ./rbac-operator /rbac-operator

ENTRYPOINT ["/rbac-operator"]
