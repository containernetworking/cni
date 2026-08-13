FROM alpine:3.20

RUN apk add --no-cache curl jq

RUN adduser -D -s /bin/sh appuser

COPY entrypoint.sh /entrypoint.sh

USER appuser

ENTRYPOINT ["/entrypoint.sh"]
