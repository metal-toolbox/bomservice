FROM alpine:3.19.1

ENTRYPOINT ["/usr/sbin/bomservice"]

COPY bomservice /usr/sbin/bomservice
RUN chmod +x /usr/sbin/bomservice
