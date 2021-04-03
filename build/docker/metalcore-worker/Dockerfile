# choose "almost" anything you like as your base (glibc based distros are easiest to use out-of-the-box)
FROM registry.access.redhat.com/ubi8-minimal

RUN microdnf update && microdnf clean all

COPY metalcore /usr/local/bin/

WORKDIR /app

# copy your service binary and deps here, or mount them via volume
COPY service .

# ENV parameters
ENV QUEUEHOST=localhost \
    QUEUEPORT=11300 \
    TASKQUEUENAME=tasks \
    RESULTQUEUENAME=results \
    CONNSHAREFACTOR=1 \
    CPULIMIT="" \
    ERRORLIMIT="" \
    LOGLEVEL=info \
    LOGPRETTYPRINT=off \
    SERVICEPATH=/app/service

VOLUME /app

ENTRYPOINT ["metalcore"]