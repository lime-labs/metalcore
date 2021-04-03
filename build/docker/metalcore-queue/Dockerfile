##################### BUILDER #####################
FROM registry.access.redhat.com/ubi8-minimal as builder

RUN microdnf update && microdnf clean all
RUN microdnf install git gcc make && microdnf clean all
RUN git clone https://github.com/beanstalkd/beanstalkd.git /tmp/beanstalkd && cd /tmp/beanstalkd && make

###################### IMAGE ######################
FROM registry.access.redhat.com/ubi8-minimal

RUN microdnf update && microdnf clean all

COPY --from=builder /tmp/beanstalkd/beanstalkd /usr/local/bin/

# can be used to store binlogs for persistency
WORKDIR /queue

VOLUME /queue

EXPOSE 11300

ENTRYPOINT ["beanstalkd"]