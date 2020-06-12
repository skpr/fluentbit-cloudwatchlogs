FROM previousnext/golang:1.13 as build
ADD . /go/src/github.com/skpr/fluentbit-cloudwatchlogs
WORKDIR /go/src/github.com/skpr/fluentbit-cloudwatchlogs
ENV CGO_ENABLED=0
RUN go build -ldflags "-extldflags -static" -o bin/fluentbit-cloudwatchlogs -a github.com/skpr/fluentbit-cloudwatchlogs/cmd/fluentbit-cloudwatchlogs

FROM scratch
COPY --from=build /go/src/github.com/skpr/fluentbit-cloudwatchlogs/bin/fluentbit-cloudwatchlogs /usr/local/bin/fluentbit-cloudwatchlogs
CMD ["fluentbit-cloudwatchlogs"]
