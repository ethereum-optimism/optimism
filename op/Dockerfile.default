FROM debian:bookworm-20240812-slim
ENTRYPOINT ["/op"]
COPY op /op

# Install ca-certificates so that HTTPS requests work
RUN apt-get update && apt-get install -y ca-certificates

# Symlink onto the PATH
RUN ln -s /op /usr/local/bin/op
