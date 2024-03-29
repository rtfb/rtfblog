FROM node:20.9.0 AS js-builder

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends -y \
    curl \
    git \
    make \
    wget \
    && wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add - \
    && sh -c 'echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list' \
    && apt-get update \
    && apt-get install --no-install-recommends -y \
        google-chrome-stable fonts-freefont-ttf libxss1 \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g grunt-cli bower browserify

WORKDIR /code

COPY package.json package.json
COPY package-lock.json package-lock.json
COPY bower.json bower.json

RUN npm install
RUN bower install --config.interactive=false

COPY js js
COPY Gruntfile.js .
COPY Makefile .

RUN grunt
RUN mkdir -p build/static/css
RUN make jsbundles

FROM ubuntu:22.04 AS dev-env

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends -y \
    build-essential \
    ca-certificates \
    curl \
    git \
    gnupg \
    jq \
    libpq-dev \
    make \
    postgresql \
    software-properties-common \
    sqlite3 \
    sudo \
    vim \
    wget

RUN add-apt-repository ppa:longsleep/golang-backports \
    && apt update \
    && apt install golang-go -y

RUN adduser --disabled-password --gecos '' rtfb --shell /bin/bash
RUN adduser rtfb sudo
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

RUN mkdir -p /app
RUN chown -R rtfb:rtfb /app
RUN mkdir -p /home/rtfb/dev
RUN chown -R rtfb:rtfb /home/rtfb/dev

USER rtfb
WORKDIR /home/rtfb/dev

# Make a copy outside of dev so that we don't overshadow it with a mount when
# we shell into the container. The 'make all' will copy that into the actual
# build output
COPY --from=js-builder /code/build /home/rtfb/jsbuild

COPY --chown=rtfb:rtfb go.mod go.sum ./

RUN go mod download
RUN go mod verify

RUN go install -tags 'postgres,sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.2 \
    && go install github.com/mattn/go-sqlite3

ENV PATH="$PATH:/home/rtfb/go/bin"
ENV GOPATH="/home/rtfb/go"

# Call entrypoint.sh on ENTRYPOINT in order to fixup permissions for use in the
# container. CMD will be passed to entrypoint.sh, which will call it. These
# shenanigans are needed to make my Dockerfile work on GH Actions. Solution
# adapted from here: https://stackoverflow.com/a/39398511
ENTRYPOINT ["/bin/bash", "scripts/entrypoint.sh"]
CMD ["make all"]
