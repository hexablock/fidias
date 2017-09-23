#!/bin/bash
#
# start-cluster.sh starts a single node, then starts additional peers joining them to one
# of the existing peers.
#
# Usage:
#   ./scripts/start-cluster.sh 4 -debug
#
set -e

COUNT=${1:-2}
BIN="fidiasd"

[[ ! -e ${BIN} ]] && { echo "$BIN not found!"; exit 1; }

sb="54321"
hport="7700"
bport="42100"

args="$@"
DEFAULT_ARGS="${args[@]:1} -bind-addr 127.0.0.1:"
CMD="${BIN} ${DEFAULT_ARGS}"

./${CMD}$sb -http-addr 127.0.0.1:$hport -data-dir "./tmp/127.0.0.1:$sb" -blox-addr 127.0.0.1:$bport &

for i in `seq $COUNT`; do
    sleep 1;
    hp=`expr $hport \+ $i`
    jp=`expr $sb \+ $i`
    bp=`expr $bport \+ $i`
    ./${CMD}$jp -http-addr 127.0.0.1:$hp -join 127.0.0.1:$sb -data-dir "./tmp/127.0.0.1:$jp" -blox-addr 127.0.0.1:$bp &
done

trap "{ pkill -9 ${BIN}; }" SIGINT SIGTERM
wait
