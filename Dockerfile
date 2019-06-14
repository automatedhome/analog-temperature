FROM arm32v7/golang:stretch

COPY qemu-arm-static /usr/bin/
WORKDIR /go/src/github.com/automatedhome/analog-temperature
COPY . .
RUN make build

FROM arm32v7/busybox:1.30-glibc

COPY --from=0 /go/src/github.com/automatedhome/analog-temperature/analog-temperature /usr/bin/analog-temperature

ENTRYPOINT [ "/usr/bin/analog-temperature" ]
