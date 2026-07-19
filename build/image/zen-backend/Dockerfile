ARG BASE=ubuntu:22.04
FROM ${BASE}

ARG NAME=zen-backend
ARG CONFIG_TGT
ARG CONFIG_DIR=/etc/zen-backend

RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /app /data /var/log/${NAME} ${CONFIG_DIR}

WORKDIR /app

COPY bin/${NAME} /app/${NAME}
COPY ${CONFIG_TGT}/${NAME}/config.yaml ${CONFIG_DIR}/config.yaml
COPY ${CONFIG_TGT}/${NAME}/migrations ${CONFIG_DIR}/migrations

EXPOSE 8080
EXPOSE 6060

ENTRYPOINT ["/app/zen-backend"]
CMD ["daemon", "-c", "/etc/zen-backend/config.yaml"]
