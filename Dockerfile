FROM ubuntu:22.04

# the following packages are needed by chromium, which is installed by the
# puppeteer:
# libnss3, libgbm-dev, libasound2

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    curl \
    git \
    golang \
    jq \
    libpq-dev \
    make \
    npm \
    postgresql \
    sqlite3 \
    sudo \
    vim \
    wget

RUN adduser --disabled-password --gecos '' rtfb
RUN adduser rtfb sudo
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

RUN mkdir -p /app
RUN chown -R rtfb:rtfb /app
RUN mkdir -p /home/rtfb/dev
RUN chown -R rtfb:rtfb /home/rtfb/dev

USER rtfb

# Set up non-system npm global prefix following the first approach in this
# overview: https://2ality.com/2022/06/global-npm-install-alternatives.html
RUN mkdir /home/rtfb/.npm-global && \
    npm config set prefix '/home/rtfb/.npm-global'

RUN npm install -g grunt-cli bower browserify

RUN go install -tags 'postgres,sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.2 && \
    go install github.com/go-bindata/go-bindata/go-bindata@latest

ENV PATH="$PATH:/home/rtfb/.npm-global/bin"
ENV PATH="$PATH:/home/rtfb/go/bin"
ENV GOPATH="/home/rtfb/go"

WORKDIR /home/rtfb/dev

ENTRYPOINT make all
