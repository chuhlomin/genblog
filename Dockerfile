FROM golang:1.17 as builder

WORKDIR /go/src/app
ADD . /go/src/app

RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOPROXY=https://athens.chuhlomin.com \
    go build -mod=readonly -a -installsuffix cgo \
    -o /go/bin/app .


FROM gcr.io/distroless/static:966f4bd97f611354c4ad829f1ed298df9386c2ec
# latest-amd64 -> 966f4bd97f611354c4ad829f1ed298df9386c2ec
# https://github.com/GoogleContainerTools/distroless/tree/master/base

LABEL name="genblog"
LABEL repository="http://github.com/chuhlomin/genblog"
LABEL homepage="http://github.com/chuhlomin/genblog"

LABEL maintainer="Konstantin Chukhlomin <mail@chuhlomin.com>"
LABEL com.github.actions.name="GenBlog"
LABEL com.github.actions.description="Generate a static blog from markdown files"
LABEL com.github.actions.icon="book"
LABEL com.github.actions.color="purple"

COPY --from=builder /go/bin/app /app

CMD ["/app"]
