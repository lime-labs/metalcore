FROM centos:8

WORKDIR /app

# COPY bin/metalcore-imp /usr/bin/
COPY bin/metalcore .

# copy your service binary and deps here
COPY bin/service .

# ENV parameters
ENV AMQPHOST=localhost \
    AMQPPORT=5672 \
    AMQPUSER=guest \
    AMQPPASSWORD=guest \
    TASKQUEUENAME=metalcore-tasks \
    SERVICEPATH=./service

ENTRYPOINT [ "/app/metalcore" ]
# CMD [ "set other parameters for the entrypoint here" ]