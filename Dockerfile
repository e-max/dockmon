FROM busybox
ADD ./bin/check /usr/bin/
ADD ./bin/monitor /usr/bin/monitor
ADD ./bin/listener /usr/bin/listener

ENTRYPOINT ["/usr/bin/monitor"]

