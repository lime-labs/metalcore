FROM alpine

RUN apk add --no-cache beanstalkd

EXPOSE 11300

# CMD below will set the max job size to 512 MiB
ENTRYPOINT ["/usr/bin/beanstalkd"]
CMD ["-z", "536870912"]