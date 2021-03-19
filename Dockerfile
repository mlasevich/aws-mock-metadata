FROM golang:1.14.1-alpine3.11
LABEL maintainer="Jerome Touffe-Blin <jtblin@gmail.com>"
RUN apk add --no-cache git
WORKDIR /go/src/app
COPY . .
RUN go get -d -v ./...
RUN go install -v ./...

# This image is like 13MB exported... :)
FROM alpine:3.11
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=0 /go/bin/aws-mock-metadata ./aws-mock-metadata
EXPOSE 45000
CMD ["./aws-mock-metadata", "--app-port", "45000"]
