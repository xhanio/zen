#!/bin/sh
set -e
/app/zen-backend daemon -c /etc/zen-backend/config.yaml &
/app/zen-mcp daemon -c /etc/zen-mcp/config.yaml &
exec nginx -g 'daemon off;'
