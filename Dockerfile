FROM btwiuse/arch:gopherjs

ADD . /app
WORKDIR /app
RUN make all

EXPOSE 8080
CMD ["go", "run", "./cmd/serve"]
