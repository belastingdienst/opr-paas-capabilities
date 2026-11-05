# Image for the in-cluster git server used by the e2e tests.
# Build context: test/e2e/.generated/gitserver (prepared by hack/setup.sh), which contains
# main.go, this Dockerfile and repos/capabilities.git (a bare snapshot of the working tree).
FROM docker.io/library/golang:1.26-alpine AS build
WORKDIR /src
COPY main.go .
RUN CGO_ENABLED=0 go build -o /gitserver main.go

FROM docker.io/library/alpine:3.22
# git-daemon provides git http-backend
RUN apk add --no-cache git git-daemon \
 && git config --system --add safe.directory '*'
COPY --from=build /gitserver /usr/local/bin/gitserver
COPY repos/ /srv/git/
USER 65534
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gitserver"]
