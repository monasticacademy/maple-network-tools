FROM alpine:3
WORKDIR /app
ENV SPEEDLOGGER_MACHINE=synology
ENV SPEEDLOGGER_NETWORK=ethernet
ENV SPEEDLOGGER_SERVICE_PROVIDER=vtel
RUN ls -l /app
ADD speed-logger /app/
CMD /app/speed-logger
