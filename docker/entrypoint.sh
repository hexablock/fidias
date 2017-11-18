#!/bin/sh
set -ex

NAME="fid"
DEFAULTS="-agent -gossip-bind-addr=0.0.0.0:32100 -data-bind-addr=0.0.0.0:42100 -rpc-bind-addr=0.0.0.0:8800 -http-addr=0.0.0.0:9090"

# If the first arg starts with '-', append to the defaults
if [ "${1:0:1}" = '-' ]; then
	set -- $NAME $DEFAULTS "$@"
# Default if no args are provided
elif [ $# -eq 0 ]; then
	set -- $NAME $DEFAULTS
fi

exec "$@"
