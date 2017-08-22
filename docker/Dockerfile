FROM alpine

ENV FIDS_ADV_ADDR=
ENV FIDS_HTTP_ADV_ADDR=

EXPOSE 32100
EXPOSE 7700

ADD dist/fidiasd-linux.tgz /bin/
ADD docker/entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
