FROM golang:alpine
# RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk add tzdata
RUN apk add build-base
ENV TZ=Asia/Shanghai
WORKDIR /messenger
COPY . .
# RUN go env -w GOPROXY=https://goproxy.cn,direct \
#     && CGO_ENABLED=1 go build -o ./messenger ./main.go
RUN CGO_ENABLED=1 go build -o ./messenger ./main.go

FROM alpine:latest
# RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk add tzdata
ENV TZ=Asia/Shanghai
WORKDIR /messenger
COPY --from=0 /messenger/messenger .
COPY --from=0 /messenger/web/build ./web/build
CMD [ "./messenger"]
