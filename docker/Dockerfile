FROM alpine

ENV FID_PEERS=

ENV FID_DATADIR=/data

ENV FID_GOSSIP_ADV_ADDR=
ENV FID_DATA_ADV_ADDR=
ENV FID_RPC_ADV_ADDR=

# Gossip
EXPOSE 32100/tcp 32100/udp
# Data
EXPOSE 42100/tcp 42100/udp
# RPC
EXPOSE 8800
# Gateway
EXPOSE 9090

ADD dist/fid-linux.tgz /bin/
ADD docker/entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
