#!/bin/bash
#
# common.sh contains common variables used by the other scripts
#
# Usage:
#   source ./scripts/common.sh
#

# Host
host=${FIDS_GATEWAY:-"127.0.0.1:7700"}
# URL prefixes
p_blox="v1/blox"
p_kv="v1/kv"
p_fs="v1/fs"
p_hexalog="v1/hexalog"
p_lookup="v1/lookup"
p_locate="v1/locate"

key="$1"
value="$2"

op=
if [ "${#@}" -eq 1 ]; then
  op="r"
elif [ "${#@}" -eq 2 ]; then
  op="w"
fi