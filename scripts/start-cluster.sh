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

BIN="fid"

[[ ! -e ${BIN} ]] && { echo "$BIN not found!"; exit 1; }


gossip_port="32100"
data_port="42100"
grpc_port="8800"
http_port="9090"

args="$@"
DEFAULT_ARGS="${args[@]:1} -agent -gossip-bind-addr 127.0.0.1:"

CMD="${BIN} ${DEFAULT_ARGS}"

./${CMD}${gossip_port} \
    -http-addr 127.0.0.1:${http_port} \
    -data-bind-addr 127.0.0.1:${data_port} \
    -rpc-bind-addr 127.0.0.1:${grpc_port} \
    -data-dir "./tmp/127.0.0.1:${gossip_port}" &

for i in `seq $COUNT`; do

    hp=`expr $http_port \+ $i`
    jp=`expr $gossip_port \+ $i`
    bp=`expr $data_port \+ $i`
    rp=`expr $grpc_port \+ $i`

    ./${CMD}$jp -join 127.0.0.1:$gossip_port -data-dir "./tmp/127.0.0.1:$jp" \
        -http-addr 127.0.0.1:$hp  \
        -rpc-bind-addr 127.0.0.1:$rp \
        -data-bind-addr 127.0.0.1:$bp &

done

trap "{ pkill -9 -f "${BIN} -debug -agent"; }" SIGINT SIGTERM
wait
