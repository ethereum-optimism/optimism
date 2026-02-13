FROM debian:bookworm-20240812-slim
ENTRYPOINT ["/op-up"]
COPY op-up /op-up

# Install ca-certificates so that HTTPS requests work
RUN apt-get update && apt-get install -y ca-certificates

# Symlink onto the PATH
RUN ln -s /op-up /usr/local/bin/op-up
