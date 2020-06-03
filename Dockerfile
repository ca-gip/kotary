FROM golang:latest as build
WORKDIR $GOPATH/src/github.com/ca-gip/kotary
COPY . $GOPATH/src/github.com/ca-gip/kotary
RUN make build


FROM scratch
WORKDIR /root/
COPY --from=build /go/src/github.com/ca-gip/kotary/build/kotary .
CMD ["./kotary"]