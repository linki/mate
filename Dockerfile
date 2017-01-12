FROM alpine:latest
MAINTAINER Team Teapot @ Zalando SE <team-teapot@zalando.de>

# install ca-certificates
RUN apk --update --no-cache add ca-certificates

# add binary
ADD build/linux/mate /

ENTRYPOINT ["/mate"]
