FROM docker.io/drycc/go-dev:latest AS build
ARG LDFLAGS
ADD . /app
RUN export GO111MODULE=on \
  && cd /app \
  && CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o /usr/local/bin/logger .


FROM docker.io/library/alpine:3.12

# Add logger user and group
RUN adduser \
       -s /bin/sh \
       -D  \
       -h /opt/logger \
       logger \
       logger

COPY . /
COPY --chown=logger:logger --from=build /usr/local/bin/logger /opt/logger/sbin/logger

USER logger

CMD ["/opt/logger/sbin/logger"]
EXPOSE 1514 8088
