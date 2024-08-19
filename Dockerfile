FROM scratch

WORKDIR /
COPY ./out/* ./
ENV PATH="/"
CMD ["server"]
