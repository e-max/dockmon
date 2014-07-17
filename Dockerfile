FROM google/golang
WORKDIR /gopath/src/checker
ADD ./src/checker /gopath/src/checker
RUN go get checker
RUN go get checker/check
RUN go get checker/monitor
RUN go get checker/listener
