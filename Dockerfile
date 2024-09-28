FROM lukemathwalker/cargo-chef:latest AS chef
WORKDIR /app

RUN apt-get update && apt-get -y upgrade && apt-get install -y libclang-dev pkg-config

FROM chef AS planner
COPY ./Cargo.toml ./Cargo.lock ./
COPY ./src ./src
RUN cargo chef prepare

FROM chef AS builder
COPY --from=planner /app/recipe.json .
RUN cargo chef cook --release
COPY . .
RUN cargo build --release

FROM debian:stable-slim AS runtime
WORKDIR /app
COPY --from=builder /app/target/release/rollup-boost /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/rollup-boost"]