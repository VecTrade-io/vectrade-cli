FROM gcr.io/distroless/static-debian12:nonroot
COPY vectrade /usr/bin/vectrade
ENTRYPOINT ["/usr/bin/vectrade"]
