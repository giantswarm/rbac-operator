FROM alpine:3.14.2

RUN apk add --no-cache ca-certificates

ADD ./rbac-operator /rbac-operator

ENTRYPOINT ["/rbac-operator"]
