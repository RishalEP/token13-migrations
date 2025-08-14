#FROM golang:1.23.6-alpine as builder

ARG BASE_IMAGE
FROM ${BASE_IMAGE:-golang:1.23.6-alpine} AS builder

ARG PackageName=quicknode

WORKDIR /${PackageName}

COPY . .
RUN apk add --no-cache gcc musl-dev
ENV CGO_ENABLED=1
ENV GOFLAGS="-buildvcs=false"
ENV GOCACHE=/root/.cache/go-build

RUN ls
RUN --mount=type=cache,target=/go/pkg/mod \
    go work sync
RUN --mount=type=cache,target="/go/pkg/mod/" \
    --mount=type=cache,target="/root/.cache/go-build" \
    go build ${PackageName}


FROM alpine:latest

RUN apk --no-cache add tzdata  && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

ARG AppName=quicknode
ARG PackageName=quicknode
ARG AppDir=app
ENV APP=${AppName}
ENV APPDIR=${AppDir}

WORKDIR /app

COPY --from=builder /${PackageName}/${AppName}  ${AppName}

ENV apollo_addr=config.token13.net
ENV apollo_app_id=quicknode
ENV apollo_cluster=dev
ENV apollo_secret=ddd1d01daba840e696a43c26b3aa1d51
ENV apollo_namespace=$CONFIG

EXPOSE 8800:8800

CMD /$APPDIR/$APP -c /$APPDIR/$CONFIG
