FROM alpine:3.14.0

RUN apk add --no-cache ca-certificates

ADD ./rbac-operator /rbac-operator

ENTRYPOINT ["/rbac-operator"]
