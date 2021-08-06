# Compile viu statically
# https://doc.rust-lang.org/edition-guide/rust-2018/platform-and-target-support/musl-support-for-fully-static-binaries.html
FROM rust as build-rust

RUN rustup target add x86_64-unknown-linux-musl

RUN git clone https://github.com/atanunq/viu.git \
      && cd viu/ \
      && apt update && apt-get install -y musl-tools \
      && cargo build --release --target x86_64-unknown-linux-musl


# Compile ansi-zenikanard statically
FROM golang:1.15-alpine as build-go

WORKDIR /app

COPY ./go.mod ./go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o ansi-zenikanard main.go


# Lightweight image without playwright
# Requires zenikanard cache
FROM scratch as light

COPY --from=build-rust /viu/target/x86_64-unknown-linux-musl/release/viu /bin/viu
COPY --from=build-go /app/ansi-zenikanard /bin/ansi-zenikanard

ENTRYPOINT ["/bin/ansi-zenikanard", "-cache-only"]


# Heavyweight image including playwright to launch headless browsers for zenikanard scraping
FROM mcr.microsoft.com/playwright:v1.5.1

COPY --from=build-rust /viu/target/x86_64-unknown-linux-musl/release/viu /bin/viu
COPY --from=build-go /app/ansi-zenikanard /bin/ansi-zenikanard

# Preload browsers. Necessary due to hardcoded browser versions in playwright-go
# different than the ones packaged in this image.
RUN /bin/ansi-zenikanard -playwright-install

ENTRYPOINT ["/bin/ansi-zenikanard"]
