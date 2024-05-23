#!/bin/bash
#From https://github.com/oneclickvirt/cpuTest
#2024.05.23

rm -rf /usr/bin/cpuTest
os=$(uname -s)
arch=$(uname -m)

case $os in
  Linux)
    case $arch in
      "x86_64" | "x86" | "amd64" | "x64")
        wget -O cpuTest https://github.com/oneclickvirt/cpuTest/releases/download/output/cpuTest-linux-amd64
        ;;
      "i386" | "i686")
        wget -O cpuTest https://github.com/oneclickvirt/cpuTest/releases/download/output/cpuTest-linux-386
        ;;
      "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64")
        wget -O cpuTest https://github.com/oneclickvirt/cpuTest/releases/download/output/cpuTest-linux-arm64
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
        wget -O cpuTest https://github.com/oneclickvirt/cpuTest/releases/download/output/cpuTest-darwin-amd64
        ;;
      "i386" | "i686")
        wget -O cpuTest https://github.com/oneclickvirt/cpuTest/releases/download/output/cpuTest-darwin-386
        ;;
      "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64")
        wget -O cpuTest https://github.com/oneclickvirt/cpuTest/releases/download/output/cpuTest-darwin-arm64
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
        wget -O cpuTest https://github.com/oneclickvirt/cpuTest/releases/download/output/cpuTest-freebsd-amd64
        ;;
      "i386" | "i686")
        wget -O cpuTest https://github.com/oneclickvirt/cpuTest/releases/download/output/cpuTest-freebsd-386
        ;;
      "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64")
        wget -O cpuTest https://github.com/oneclickvirt/cpuTest/releases/download/output/cpuTest-freebsd-arm64
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
        wget -O cpuTest https://github.com/oneclickvirt/cpuTest/releases/download/output/cpuTest-openbsd-amd64
        ;;
      "i386" | "i686")
        wget -O cpuTest https://github.com/oneclickvirt/cpuTest/releases/download/output/cpuTest-openbsd-386
        ;;
      "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64")
        wget -O cpuTest https://github.com/oneclickvirt/cpuTest/releases/download/output/cpuTest-openbsd-arm64
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

chmod 777 cpuTest
if [ ! -f /usr/bin/cpuTest ]; then
  mv cpuTest /usr/bin/
  cpuTest
else
  ./cpuTest
fi