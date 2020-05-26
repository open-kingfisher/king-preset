FROM golang:1.14.3 as builder
ARG NAME="king-preset"
ARG GIT_URL="https://github.com/open-kingfisher/$NAME.git"
RUN git clone $GIT_URL /$NAME && cd /$NAME && make

FROM alpine:3.10

ARG NAME="king-preset"
ENV TIME_ZONE Asia/Shanghai
RUN set -xe \
    && sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk --no-cache add tzdata \
    && echo "${TIME_ZONE}" > /etc/timezone \
    && ln -sf /usr/share/zoneinfo/${TIME_ZONE} /etc/localtime \
    && mkdir /lib64 \
    && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
COPY --from=builder /$NAME/entrypoint.sh /entrypoint.sh
COPY --from=builder /$NAME/bin/$NAME /usr/local/bin

ENTRYPOINT ["/bin/sh","/entrypoint.sh"]
