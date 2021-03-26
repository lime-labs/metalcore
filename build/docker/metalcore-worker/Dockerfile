# choose anything you like as your base
FROM centos:8

WORKDIR /app

COPY bin/metalcore .

# copy your service binary and deps here
COPY bin/service .

# ENV parameters
ENV QUEUEHOST=localhost \
    QUEUEPORT=11300 \
    TASKQUEUENAME=tasks \
    RESULTQUEUENAME=results \
    SERVICEPATH=./service

ENTRYPOINT ["/app/metalcore"]
# CMD ["set other parameters for the entrypoint here. Currently not used in metalcore IMP."]