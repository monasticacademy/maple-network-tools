FROM alpine:3
WORKDIR /app
RUN apk add ghostscript
ENV SUBSCRIPTION=print-queue
ENV PRINTER=dir:/tmp
ADD .bin/thankyou-letter-printer /app/
ADD testdata /app/testdata
CMD /app/thankyou-letter-printer
