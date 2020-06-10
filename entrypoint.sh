#!/bin/sh

[ "$TIME_ZONE" ] || TIME_ZONE="Asia/Shanghai"
[ "$ALPINE_REPO" ] || ALPINE_REPO="mirrors.aliyun.com"

sed -i "s/dl-cdn.alpinelinux.org/${ALPINE_REPO}/g" /etc/apk/repositories     
apk --no-cache add tzdata 
echo "${TIME_ZONE}" > /etc/timezone 
ln -sf /usr/share/zoneinfo/${TIME_ZONE} /etc/localtime 
mkdir /lib64 
ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2


/usr/local/bin/king-preset


