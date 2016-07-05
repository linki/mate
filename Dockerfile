FROM registry.opensource.zalan.do/stups/alpine:3.4-2

# add scm-source
ADD scm-source.json /

# add binary
ADD build/linux/mate /

ENTRYPOINT ["/mate"]
