#!/bin/bash
#
# common.sh contains common variables used by the other scripts
#
# Usage:
#   source ./scripts/common.sh
#

# Host
host=${FIDS_GATEWAY:-"127.0.0.1:9090"}
# URL prefixes
p_blox="blox"
p_kv="kv"
p_fs="v1/fs"
p_hexalog="v1/hexalog"
p_lookup="v1/lookup"
p_locate="v1/locate"
p_status="v1/status"

indent() {
    echo "${1}" | sed -e "s/^/${2}/g"
}

http_headers() {
    echo "${1}" | grep "<" | sed -e "s/< //g"
}

http_body() {
    echo "${resp}" | grep '{.*}' | jq .
}

http_get() {
	resp=`curl -L -v "${1}" 2>&1`
	http_headers "${resp}"
    http_body "${resp}"
}

key="$1"
value="$2"

op=
if [ "${#@}" -eq 1 ]; then
  op="r"
elif [ "${#@}" -eq 2 ]; then
  op="w"
fi
