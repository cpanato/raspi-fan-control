FROM gcr.io/distroless/base:debug

COPY raspi-fan-control /bin/raspi-fan-control

USER nobody

ENTRYPOINT [ "/bin/raspi-fan-control" ]
