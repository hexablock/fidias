#!/bin/sh
set -e

NAME="fidiasd"
DEFAULTS="-bind-addr=0.0.0.0:32100 -http-addr=0.0.0.0:7700"

# If the first arg starts with '-', append to the defaults
if [ "${1:0:1}" = '-' ]; then
	set -- $NAME $DEFAULTS "$@"
# Default if no args are provided
elif [ $# -eq 0 ]; then
	set -- $NAME $DEFAULTS
fi

exec "$@"
