FROM golang:1.18 as builder
 
WORKDIR /go/src/github.com/automatedhome/analog-temperature
COPY . .
RUN CGO_ENABLED=0 go build -o analog-temperature cmd/main.go

FROM busybox:glibc

COPY --from=builder /go/src/github.com/automatedhome/analog-temperature/analog-temperature /usr/bin/analog-temperature

ENTRYPOINT [ "/usr/bin/analog-temperature" ]
