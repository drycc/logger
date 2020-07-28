FROM alpine:3.12

# Add logger user and group
RUN adduser \
       -s /bin/sh \
       -D  \
       -h /opt/logger \
       logger \
       logger

COPY . /

# Fix some permission since we'll be running as a non-root user
RUN chown -R logger:logger /opt/logger

USER logger

CMD ["/opt/logger/sbin/logger"]
EXPOSE 1514 8088
