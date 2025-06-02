FROM alpine:3.22.0

ENTRYPOINT ["/usr/sbin/bomservice"]

COPY bomservice /usr/sbin/bomservice
RUN chmod +x /usr/sbin/bomservice
