FROM debian:bookworm-slim

RUN DEBIAN_FRONTEND=noninteractive apt-get update -qq && \
    apt-get install -y python3 python3-pip wget

RUN wget -q -O - https://apt.mopidy.com/mopidy.gpg | apt-key add -
RUN wget -q -O /etc/apt/sources.list.d/mopidy.list https://apt.mopidy.com/buster.list

RUN DEBIAN_FRONTEND=noninteractive apt-get update -qq && \
    apt-get install -y gstreamer1.0-plugins-bad mopidy mopidy-local mopidy-mpd

RUN pip3 install Mopidy-Tidal

# Do not use VOLUME, otherwise they get owned by root by default
#VOLUME /opt/mopidy/cache
#VOLUME /opt/mopidy/data
#VOLUME /opt/mopidy/music
#VOLUME /var/lib/mopidy/.config

RUN  usermod -u 1000 mopidy
RUN  mkdir -p      /opt/mopidy/cache /opt/mopidy/data /opt/mopidy/music
RUN  chown mopidy: /opt/mopidy/cache /opt/mopidy/data /opt/mopidy/music /var/lib/mopidy
RUN  usermod -a -G 29 mopidy

COPY entrypoint.sh /entrypoint.sh
RUN  chmod 0755    /entrypoint.sh
ENTRYPOINT "/entrypoint.sh"

EXPOSE 6600 6680

COPY mopidy.conf /etc/mopidy.conf
COPY asound.conf /etc/asound.conf
RUN  chmod 0644  /etc/mopidy.conf /etc/asound.conf

USER mopidy
