FROM ubuntu:22.04

# the following packages are needed by chromium, which is installed by the
# puppeteer:
# libnss3, libgbm-dev, libasound2

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends -y \
    build-essential \
    curl \
    git \
    gnupg \
    golang \
    jq \
    libasound2 \
    libgbm-dev \
    libnss3 \
    libpq-dev \
    make \
    npm \
    postgresql \
    sqlite3 \
    sudo \
    vim \
    wget \
    && wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add - \
    && sh -c 'echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list' \
    && apt-get update \
    && apt-get install --no-install-recommends -y \
        google-chrome-stable fonts-freefont-ttf libxss1 \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g grunt-cli bower browserify

RUN adduser --disabled-password --gecos '' rtfb --shell /bin/bash
RUN adduser rtfb sudo
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

RUN mkdir -p /app
RUN chown -R rtfb:rtfb /app
RUN mkdir -p /home/rtfb/dev
RUN chown -R rtfb:rtfb /home/rtfb/dev

USER rtfb

ENV NVM_DIR /home/rtfb/.nvm
ENV NODE_VERSION 20

WORKDIR /home/rtfb/dev

ADD --chown=rtfb:rtfb package*.json ./

# Install nvm with node and npm following this SO answer:
# https://stackoverflow.com/a/28390848
RUN curl https://raw.githubusercontent.com/creationix/nvm/v0.39.5/install.sh | bash \
    && . $NVM_DIR/nvm.sh \
    && nvm install $NODE_VERSION \
    && nvm alias default $NODE_VERSION \
    && nvm use default \
    && npm install

ADD --chown=rtfb:rtfb go.mod go.sum ./

RUN go mod download

RUN go install -tags 'postgres,sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.2 \
    && go install github.com/go-bindata/go-bindata/go-bindata@latest \
    && go install github.com/mattn/go-sqlite3

ENV PATH="$PATH:/home/rtfb/.npm-global/bin"
ENV PATH="$PATH:/home/rtfb/go/bin"
ENV GOPATH="/home/rtfb/go"

# Explicitly call bash instead of the default sh, plus source nvm.sh to switch
# to the correct node version, only then call make:
ENTRYPOINT /bin/bash -c -- "source /home/rtfb/.nvm/nvm.sh && make all"
