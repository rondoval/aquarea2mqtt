FROM golang as builder
RUN go get github.com/rondoval/aquarea2mqtt \
    && CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o /go/bin/aquarea2mqtt /go/src/github.com/rondoval/aquarea2mqtt/

FROM alpine
RUN adduser -S -D -H -h /aquarea appuser
USER appuser
COPY --from=builder /go/bin/aquarea2mqtt /aquarea/aquarea2mqtt
COPY --from=builder /go/src/github.com/rondoval/aquarea2mqtt/config.example.json /aquarea/config.json
COPY --from=builder /go/src/github.com/rondoval//aquarea2mqtt/translation.json /aquarea/translation.json
WORKDIR /aquarea
ENTRYPOINT ./aquarea2mqtt
