#!/bin/bash
#From https://github.com/oneclickvirt/cputest
#2024.08.05

rm -rf /usr/bin/geekbench*
arch=$(uname -m)
release_date="20240525"
mypwd=$(pwd)
if [ ! -d "/usr/bin/" ]; then
    mkdir /usr/bin/
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
check_ipv4_available() {
    if ! curl -s 'https://browser.geekbench.com' --connect-timeout 5 >/dev/null; then
        echo -e "No IPV4 network, can't test with geekbench, browser.geekbench.com only has IPv4, does not support IPv6, forcing to test with no result."
        exit 1
    fi
}
check_ipv4_available

# 除了 geekbench 4 , 更高版本的 geekbench需要本机至少有 1 GB 内存
mem=$(free -m | awk '/Mem/{print $2}')
swap=$(free -m | awk '/Swap/{print $2}')
ms=$((mem + old_swap))
if [ "$mem" -ge "1024" ]; then
    echo "After judgment, the memory of this machine is greater than 1G, which meets the GB5/GB6 test conditions"
elif [ "$old_ms" -ge "1280" ]; then
    echo "After judgment, the total amount of RAM plus Swap of this machine is more than 1.25G, which meets the GB5/GB6 test conditions."
else
    echo "After judgment, the total memory plus Swap of this machine is less than 1.25G, switch to GB4 for testing."
    gbv="gb4"
fi

check_cdn() {
  local o_url=$1
  for cdn_url in "${cdn_urls[@]}"; do
    if curl -sL -k "$cdn_url$o_url" --max-time 6 | grep -q "success" >/dev/null 2>&1; then
      export cdn_success_url="$cdn_url"
      return
    fi
    sleep 0.5
  done
  export cdn_success_url=""
}

check_cdn_file() {
  check_cdn "https://raw.githubusercontent.com/spiritLHLS/ecs/main/back/test"
  if [ -n "$cdn_success_url" ]; then
    echo "CDN available, using CDN"
  else
    echo "No CDN available, no use CDN"
  fi
}

cdn_urls=("https://cdn0.spiritlhl.top/" "http://cdn3.spiritlhl.net/" "http://cdn1.spiritlhl.net/" "http://cdn2.spiritlhl.net/")
check_cdn_file

download_file() {
    local url="$1"
    local output="$2"
    
    if ! wget -O "$output" "$url"; then
        echo "wget failed, trying curl..."
        if ! curl -L -o "$output" "$url"; then
            echo "Both wget and curl failed. Unable to download the file."
            return 1
        fi
    fi
    return 0
}

# 下载对应文件
case $gbv in
  gb4)
    case $arch in
      "x86_64" | "x86" | "amd64" | "x64")
        download_file "${cdn_success_url}https://github.com/oneclickvirt/cputest/releases/download/${release_date}/Geekbench-4.4.4-Linux.tar.gz" "/usr/bin/geekbench.tar.gz"
        cd /usr/bin >/dev/null 2>&1
        chmod 777 geekbench.tar.gz
        tar -xvf geekbench.tar.gz
        mv Geekbench-4.4.4-Linux/geekbench4 geekbench
        mv Geekbench-4.4.4-Linux/geekbench_x86_64 geekbench_x86_64
        mv Geekbench-4.4.4-Linux/geekbench.plar geekbench.plar
        rm -rf Geekbench-4.4.4-Linux
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
        download_file "${cdn_success_url}https://github.com/oneclickvirt/cputest/releases/download/${release_date}/Geekbench-5.5.1-Linux.tar.gz" "/usr/bin/geekbench.tar.gz"
        cd /usr/bin >/dev/null 2>&1
        chmod 777 geekbench.tar.gz
        tar -xvf geekbench.tar.gz
        mv Geekbench-5.5.1-Linux/geekbench5 geekbench
        mv Geekbench-5.5.1-Linux/geekbench_x86_64 geekbench_x86_64
        mv Geekbench-5.5.1-Linux/geekbench.plar geekbench.plar
        rm -rf Geekbench-5.5.1-Linux
        cd $mypwd >/dev/null 2>&1
        ;;
      "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64")
        download_file "${cdn_success_url}https://github.com/oneclickvirt/cputest/releases/download/${release_date}/Geekbench-5.5.1-LinuxARMPreview.tar.gz" "/usr/bin/geekbench.tar.gz"
        cd /usr/bin >/dev/null 2>&1
        chmod 777 geekbench.tar.gz
        tar -xvf geekbench.tar.gz
        mv Geekbench-5.5.1-LinuxARMPreview/geekbench5 geekbench
        mv Geekbench-5.5.1-LinuxARMPreview/geekbench_aarch64 geekbench_aarch64
        mv Geekbench-5.5.1-LinuxARMPreview/geekbench_armv7 geekbench_armv7
        mv Geekbench-5.5.1-LinuxARMPreview/geekbench.plar geekbench.plar
        rm -rf Geekbench-5.5.1-LinuxARMPreview
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
        download_file "${cdn_success_url}https://github.com/oneclickvirt/cputest/releases/download/${release_date}/Geekbench-6.3.0-Linux.tar.gz" "/usr/bin/geekbench.tar.gz"
        cd /usr/bin >/dev/null 2>&1
        chmod 777 geekbench.tar.gz
        tar -xvf geekbench.tar.gz
        mv Geekbench-6.3.0-Linux/geekbench6 geekbench
        mv Geekbench-6.3.0-Linux/geekbench_x86_64 geekbench_x86_64
        mv Geekbench-6.3.0-Linux/geekbench.plar geekbench.plar
        rm -rf Geekbench-6.3.0-Linux
        cd $mypwd >/dev/null 2>&1
        ;;
      "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64")
        download_file "${cdn_success_url}https://github.com/oneclickvirt/cputest/releases/download/${release_date}/Geekbench-6.3.0-LinuxARMPreview.tar.gz" "/usr/bin/geekbench.tar.gz"
        cd /usr/bin >/dev/null 2>&1
        chmod 777 geekbench.tar.gz
        tar -xvf geekbench.tar.gz
        mv Geekbench-6.3.0-LinuxARMPreview/geekbench6 geekbench
        mv Geekbench-6.3.0-LinuxARMPreview/geekbench_aarch64 geekbench_aarch64
        mv Geekbench-6.3.0-LinuxARMPreview/geekbench_armv7 geekbench_armv7
        mv Geekbench-6.3.0-LinuxARMPreview/geekbench.plar geekbench.plar
        rm -rf Geekbench-6.3.0-LinuxARMPreview
        cd $mypwd >/dev/null 2>&1
        ;;
      *)
        echo "Unsupported architecture: $arch"
        exit 1
        ;;
    esac
    ;;
esac

if [ -f /usr/bin/geekbench ]; then 
  chmod 777 /usr/bin/geekbench
  /usr/bin/geekbench --version
  if [ $? -ne 0 ]; then
    echo "Geekbench failed to check the version, please leave an error message in the repository's issues."
  fi
  rm -rf /usr/bin/geekbench.tar.gz
else
  echo "Geekbench failed to download, please leave an error message in the repository's issues."
fi
