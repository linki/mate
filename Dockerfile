FROM registry.opensource.zalan.do/stups/alpine:UPSTREAM
MAINTAINER Team Teapot @ Zalando SE <team-teapot@zalando.de>

# add scm-source
ADD scm-source.json /

# add binary
ADD build/linux/mate /

ENTRYPOINT ["/mate"]
