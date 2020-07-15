FROM postgres:latest



COPY ./scripts/* /docker-entrypoint-initdb.d/

