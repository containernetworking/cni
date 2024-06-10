FROM alpine:3.20

RUN apk add --no-cache curl jq

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
