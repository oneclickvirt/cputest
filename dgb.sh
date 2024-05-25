#!/bin/bash
#From https://github.com/oneclickvirt/cpuTest
#2024.05.25


rm -rf /tmp/geekbench
arch=$(uname -m)
release_date="20240525"
mypwd=$(pwd)
if [ ! -d "/tmp" ]; then
    mkdir /tmp
fi
if ! command -v wget >/dev/null 2>&1; then
    echo "The wget command is not detected, please download it before executing this script."
fi
if ! command -v tar >/dev/null 2>&1; then
    echo "The tar command is not detected, please download it before executing this script."
fi
if [ "$1" != "-v" ]; then
  echo "Error: the -v option must be used"
  exit 1
fi
if [ -z "$2" ]; then
  echo "Error: a value must be provided (gb4, gb5 or gb6)"
  exit 1
fi
case "$2" in
  gb4|gb5|gb6)
    gbv="$2"
    ;;
  *)
    echo "Error: Invalid value. Must be gb4, gb5 or gb6"
    exit 1
    ;;
esac

# 检测本机是否存在IPV4网络，不存在时无法使用 geekbench 进行测试
# 除了 geekbench 4 , 更高版本的 geekbench需要本机至少有 1 GB 内存

# 下载对应文件
case $gbv in
  gb4)
    case $arch in
      "x86_64" | "x86" | "amd64" | "x64")
        wget -O /tmp/geekbench.tar.gz https://github.com/oneclickvirt/cpuTest/releases/download/${release_date}/Geekbench-4.4.4-Linux.tar.gz
        cd /tmp >/dev/null 2>&1
        chmod 777 geekbench.tar.gz
        tar -xvf geekbench.tar.gz
        mv Geekbench-4.4.4-Linux /tmp/geekbench
        cd $mypwd >/dev/null 2>&1
        ;;
      *)
        echo "Unsupported architecture: $arch"
        exit 1
        ;;
    esac
    ;;
  gb5)
    case $arch in
      "x86_64" | "x86" | "amd64" | "x64")
        wget -O /tmp/geekbench.tar.gz https://github.com/oneclickvirt/cpuTest/releases/download/${release_date}/Geekbench-5.5.1-Linux.tar.gz
        cd /tmp >/dev/null 2>&1
        chmod 777 geekbench.tar.gz
        tar -xvf geekbench.tar.gz
        mv Geekbench-5.5.1-Linux /tmp/geekbench
        cd $mypwd >/dev/null 2>&1
        ;;
      "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64")
        wget -O /tmp/geekbench.tar.gz https://github.com/oneclickvirt/cpuTest/releases/download/${release_date}/Geekbench-5.5.1-LinuxARMPreview.tar.gz
        cd /tmp >/dev/null 2>&1
        chmod 777 geekbench.tar.gz
        tar -xvf geekbench.tar.gz
        mv Geekbench-5.5.1-LinuxARMPreview /tmp/geekbench
        cd $mypwd >/dev/null 2>&1
        ;;
      *)
        echo "Unsupported architecture: $arch"
        exit 1
        ;;
    esac
    ;;
  gb6)
    case $arch in
      "x86_64" | "x86" | "amd64" | "x64")
        wget -O /tmp/geekbench.tar.gz https://github.com/oneclickvirt/cpuTest/releases/download/${release_date}/Geekbench-6.3.0-Linux.tar.gz
        cd /tmp >/dev/null 2>&1
        chmod 777 geekbench.tar.gz
        tar -xvf geekbench.tar.gz
        mv Geekbench-6.3.0-Linux /tmp/geekbench
        cd $mypwd >/dev/null 2>&1
        ;;
      "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64")
        wget -O /tmp/geekbench.tar.gz https://github.com/oneclickvirt/cpuTest/releases/download/${release_date}/Geekbench-6.3.0-LinuxARMPreview.tar.gz
        cd /tmp >/dev/null 2>&1
        chmod 777 geekbench.tar.gz
        tar -xvf geekbench.tar.gz
        mv Geekbench-6.3.0-LinuxARMPreview /tmp/geekbench
        cd $mypwd >/dev/null 2>&1
        ;;
      *)
        echo "Unsupported architecture: $arch"
        exit 1
        ;;
    esac
    ;;
esac

if [ -d /tmp/geekbench ]; then
  for file in /tmp/geekbench/geekbench[0-9]*; do
    if [ -f "$file" ]; then
      chmod 777 "$file"
      target="/tmp/geekbench/geekbench"
      mv "$file" "$target"
    fi
  done
fi
if [ -f /tmp/geekbench/geekbench ]; then 
  chmod 777 /tmp/geekbench/geekbench
  /tmp/geekbench/geekbench --version
  if [ $? -ne 0 ]; then
    echo "Geekbench failed to check the version, please leave an error message in the repository's issues."
  fi
  rm -rf /tmp/geekbench.tar.gz
else
  echo "Geekbench failed to download, please leave an error message in the repository's issues."
fi