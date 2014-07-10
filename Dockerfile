FROM ubuntu
ADD ./bin/regs /regs
ENTRYPOINT ["/regs"]

