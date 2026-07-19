ARG BASE=ubuntu:22.04
FROM ${BASE}

ARG NAME=zen-mcp
ARG CONFIG_TGT
ARG CONFIG_DIR=/etc/zen-mcp

RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /app /var/log/${NAME} ${CONFIG_DIR}

WORKDIR /app

COPY bin/${NAME} /app/${NAME}
COPY ${CONFIG_TGT}/${NAME}/config.yaml ${CONFIG_DIR}/config.yaml

EXPOSE 8081
EXPOSE 6061

ENTRYPOINT ["/app/zen-mcp"]
CMD ["daemon", "-c", "/etc/zen-mcp/config.yaml"]
