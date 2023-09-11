FROM alpine:3.18.0

ENTRYPOINT ["/usr/sbin/hollow-bomservice"]

COPY hollow-bomservice /usr/sbin/hollow-bomservice
RUN chmod +x /usr/sbin/hollow-bomservice
