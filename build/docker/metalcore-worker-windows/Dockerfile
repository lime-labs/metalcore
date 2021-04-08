# FROM mcr.microsoft.com/windows/servercore:ltsc2019
FROM mcr.microsoft.com/windows/nanoserver:20H2

RUN mkdir c:\\metalcore

COPY metalcore.exe c:\\metalcore\\.

WORKDIR c:\\app

# copy your service binary and deps here, or mount them via volume
COPY service.exe .

# ENV parameters
ENV QUEUEHOST=localhost \
    QUEUEPORT=11300 \
    TASKQUEUENAME=tasks \
    RESULTQUEUENAME=results \
    CONNSHAREFACTOR="" \
    CPULIMIT="" \
    ERRORLIMIT="" \
    LOGLEVEL=info \
    LOGPRETTYPRINT=off \
    SERVICEPATH=c:\\app\\service.exe

CMD ["c:\\metalcore\\metalcore.exe"]