FROM alpine:latest
LABEL authors "Darknet Villain <supervillain@riseup.net>"

ENV HOME_DIR="/home/xd"
ENV XD_HOME="$HOME_DIR/data"

RUN mkdir -p "$HOME_DIR" "$XD_HOME" \
    && adduser -S -h "$HOME_DIR" xd \
    && chown -R xd:nobody "$HOME_DIR" \
    && chmod -R 700 "$XD_HOME"

RUN apk --no-cache add go build-base git yarn \
    && git clone https://github.com/majestrate/XD /root/XD \
    && cd /root/XD \
    && make \
    && mv /root/XD/XD "$XD_HOME" \
    && chown xd "$XD_HOME"/XD && chmod +x "$XD_HOME"/XD \
    && ln -s "$XD_HOME"/XD "$XD_HOME"/xd-cli \
    && rm -rf /root/XD && apk --purge del go build-base git yarn

EXPOSE 1188

VOLUME "$XD_HOME"
WORKDIR "$XD_HOME"
USER xd

CMD ./XD torrents.ini
