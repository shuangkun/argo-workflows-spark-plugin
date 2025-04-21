FROM golang:1.23 AS builder

WORKDIR /go/src/github.com/shuangkun/argo-workflows-spark-plugin
COPY . /go/src/github.com/shuangkun/argo-workflows-spark-plugin
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags "-w -s" -o argo-spark-plugin main.go

FROM alpine:3.10
COPY --from=builder /go/src/github.com/shuangkun/argo-workflows-spark-plugin/argo-spark-plugin /usr/bin/argo-spark-plugin
RUN chmod +x /usr/bin/argo-spark-plugin
CMD ["argo-spark-plugin"]

