FROM alpine:latest
RUN apk update && apk add netcat-openbsd
COPY test/test.sh /
RUN chmod +x test.sh
ENTRYPOINT ["/test.sh"]