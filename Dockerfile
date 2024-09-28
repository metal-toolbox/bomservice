FROM alpine:3.20.3

ENTRYPOINT ["/usr/sbin/bomservice"]

COPY bomservice /usr/sbin/bomservice
RUN chmod +x /usr/sbin/bomservice
