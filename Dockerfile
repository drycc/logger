FROM registry.drycc.cc/drycc/go-dev:latest AS build
ARG LDFLAGS
ADD . /workspace
RUN export GO111MODULE=on \
  && cd /workspace \
  && CGO_ENABLED=0 init-stack go build -ldflags "${LDFLAGS}" -o /usr/local/bin/logger .


FROM registry.drycc.cc/drycc/base:bookworm

ENV DRYCC_UID=1001 \
  DRYCC_GID=1001 \
  DRYCC_HOME_DIR=/opt/logger

RUN groupadd drycc --gid ${DRYCC_GID} \
  && useradd drycc -u ${DRYCC_UID} -g ${DRYCC_GID} -s /bin/bash -m -d ${DRYCC_HOME_DIR}

COPY . /
COPY --chown=${DRYCC_UID}:${DRYCC_GID} --from=build /usr/local/bin/logger /opt/logger/bin/logger

USER ${DRYCC_UID}

CMD ["/opt/logger/bin/logger"]
EXPOSE 1514 8088
