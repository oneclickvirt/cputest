#!/bin/bash
#From https://github.com/oneclickvirt/cputest
#2024.06.24

rm -rf /usr/bin/cputest
os=$(uname -s)
arch=$(uname -m)

case $os in
  Linux)
    case $arch in
      "x86_64" | "x86" | "amd64" | "x64")
        wget -O cputest https://github.com/oneclickvirt/cputest/releases/download/output/cputest-linux-amd64
        ;;
      "i386" | "i686")
        wget -O cputest https://github.com/oneclickvirt/cputest/releases/download/output/cputest-linux-386
        ;;
      "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64")
        wget -O cputest https://github.com/oneclickvirt/cputest/releases/download/output/cputest-linux-arm64
        ;;
      *)
        echo "Unsupported architecture: $arch"
        exit 1
        ;;
    esac
    ;;
  Darwin)
    case $arch in
      "x86_64" | "x86" | "amd64" | "x64")
        wget -O cputest https://github.com/oneclickvirt/cputest/releases/download/output/cputest-darwin-amd64
        ;;
      "i386" | "i686")
        wget -O cputest https://github.com/oneclickvirt/cputest/releases/download/output/cputest-darwin-386
        ;;
      "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64")
        wget -O cputest https://github.com/oneclickvirt/cputest/releases/download/output/cputest-darwin-arm64
        ;;
      *)
        echo "Unsupported architecture: $arch"
        exit 1
        ;;
    esac
    ;;
  FreeBSD)
    case $arch in
      amd64)
        wget -O cputest https://github.com/oneclickvirt/cputest/releases/download/output/cputest-freebsd-amd64
        ;;
      "i386" | "i686")
        wget -O cputest https://github.com/oneclickvirt/cputest/releases/download/output/cputest-freebsd-386
        ;;
      "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64")
        wget -O cputest https://github.com/oneclickvirt/cputest/releases/download/output/cputest-freebsd-arm64
        ;;
      *)
        echo "Unsupported architecture: $arch"
        exit 1
        ;;
    esac
    ;;
  OpenBSD)
    case $arch in
      amd64)
        wget -O cputest https://github.com/oneclickvirt/cputest/releases/download/output/cputest-openbsd-amd64
        ;;
      "i386" | "i686")
        wget -O cputest https://github.com/oneclickvirt/cputest/releases/download/output/cputest-openbsd-386
        ;;
      "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64")
        wget -O cputest https://github.com/oneclickvirt/cputest/releases/download/output/cputest-openbsd-arm64
        ;;
      *)
        echo "Unsupported architecture: $arch"
        exit 1
        ;;
    esac
    ;;
  *)
    echo "Unsupported operating system: $os"
    exit 1
    ;;
esac

chmod 777 cputest
if [ ! -f /usr/bin/cputest ]; then
  mv cputest /usr/bin/
  cputest
else
  ./cputest
fi
