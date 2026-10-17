FROM debian:trixie-slim AS build

ARG RTORRENT_VERSION=v0.16.21

RUN apt-get update && apt-get install -y --no-install-recommends \
        autoconf \
        automake \
        build-essential \
        ca-certificates \
        git \
        libcurl4-openssl-dev \
        libssl-dev \
        libtool \
        pkg-config \
        zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

RUN git clone --depth 1 --branch "${RTORRENT_VERSION}" \
        https://github.com/rakshasa/libtorrent /usr/src/libtorrent \
    && cd /usr/src/libtorrent \
    && autoreconf -fiv \
    && ./configure --disable-static \
    && make -j"$(nproc)" \
    && make install \
    && ldconfig

RUN git clone --depth 1 --branch "${RTORRENT_VERSION}" \
        https://github.com/rakshasa/rtorrent /usr/src/rtorrent \
    && cd /usr/src/rtorrent \
    && autoreconf -fiv \
    && PKG_CONFIG_PATH=/usr/local/lib/pkgconfig \
       ./configure --with-xmlrpc-tinyxml2 --without-ncurses \
    && make -j"$(nproc)" \
    && make install

FROM debian:trixie-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        libcurl4 \
        libssl3 \
        zlib1g \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home --uid 1000 --user-group rtorrent \
    && mkdir -p /downloads /session \
    && chown -R rtorrent:rtorrent /downloads /session

COPY --from=build /usr/local/bin/rtorrent /usr/local/bin/rtorrent
COPY --from=build /usr/local/lib/libtorrent.so* /usr/local/lib/
RUN ldconfig

COPY rtorrent.rc /etc/rtorrent/rtorrent.rc

USER rtorrent
EXPOSE 5000/tcp

CMD ["rtorrent", "-o", "system.daemon.set=true", "-o", "import=/etc/rtorrent/rtorrent.rc"]
