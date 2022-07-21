FROM alpine:3
WORKDIR /app
RUN apk add ghostscript
ENV SUBSCRIPTION=print-queue
ENV PRINTER=dir:/tmp
ADD .bin/print-demon /app/
ADD testdata /app/testdata
CMD /app/print-demon
