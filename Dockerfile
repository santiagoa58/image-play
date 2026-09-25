# syntax=docker/dockerfile:1

# GoCV needs OpenCV headers and pkg-config metadata at build time.
FROM golang:1.26.3-trixie AS toolchain

# Drop apt's package indexes in this layer to keep the dev/build image smaller.
RUN apt-get update \
    && apt-get install -y --no-install-recommends libopencv-dev \
    && rm -rf /var/lib/apt/lists/*

FROM toolchain AS compile

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/mosaic ./cmd/mosaic

# The CLI only needs the OpenCV shared libraries at runtime.
FROM debian:trixie-slim AS runtime

# Drop apt's package indexes in this layer to keep the deployed image smaller.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        libopencv-calib3d410 \
        libopencv-core410 \
        libopencv-dnn410 \
        libopencv-features2d410 \
        libopencv-flann410 \
        libopencv-highgui410 \
        libopencv-imgcodecs410 \
        libopencv-imgproc410 \
        libopencv-objdetect410 \
        libopencv-photo410 \
        libopencv-video410 \
        libopencv-videoio410 \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /work \
    && chown 65532:65532 /work

COPY --from=compile /out/mosaic /usr/local/bin/mosaic
WORKDIR /work
# Match the writable /work directory without adding an account to the image.
USER 65532:65532
ENTRYPOINT ["mosaic"]
