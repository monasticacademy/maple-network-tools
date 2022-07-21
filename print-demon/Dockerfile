FROM alpine:3
WORKDIR /app
RUN apk add ghostscript
ENV SUBSCRIPTION=print-queue
ENV PRINTER=ipp://192.168.88.133
ADD .bin/print-demon /app/
ADD testdata /app/testdata
CMD /app/print-demon
