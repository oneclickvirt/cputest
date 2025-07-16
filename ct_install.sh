#!/bin/bash
#From https://github.com/oneclickvirt/cputest
#2025.07.16

rm -rf /usr/bin/cputest
rm -rf cputest
os=$(uname -s)
arch=$(uname -m)

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

get_cputest_url() {
    local os="$1"
    local arch="$2"
    case $os in
        Linux)
            case $arch in
                "x86_64" | "x86" | "amd64" | "x64") echo "cputest-linux-amd64" ;;
                "i386" | "i686") echo "cputest-linux-386" ;;
                "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64") echo "cputest-linux-arm64" ;;
                "s390x" | "s390") echo "cputest-linux-s390x" ;;
                *) return 1 ;;
            esac
            ;;
        Darwin)
            case $arch in
                "x86_64" | "x86" | "amd64" | "x64") echo "cputest-darwin-amd64" ;;
                "i386" | "i686") echo "cputest-darwin-386" ;;
                "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64") echo "cputest-darwin-arm64" ;;
                *) return 1 ;;
            esac
            ;;
        FreeBSD)
            case $arch in
                amd64) echo "cputest-freebsd-amd64" ;;
                "i386" | "i686") echo "cputest-freebsd-386" ;;
                "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64") echo "cputest-freebsd-arm64" ;;
                "s390x" | "s390") echo "cputest-freebsd-s390x" ;;
                *) return 1 ;;
            esac
            ;;
        OpenBSD)
            case $arch in
                amd64) echo "cputest-openbsd-amd64" ;;
                "i386" | "i686") echo "cputest-openbsd-386" ;;
                "armv7l" | "armv8" | "armv8l" | "aarch64" | "arm64") echo "cputest-openbsd-arm64" ;;
                "s390x" | "s390") echo "cputest-openbsd-s390x" ;;
                *) return 1 ;;
            esac
            ;;
        *) return 1 ;;
    esac
}

cputest_filename=$(get_cputest_url "$os" "$arch")
if [ -z "$cputest_filename" ]; then
    echo "Unsupported operating system ($os) or architecture ($arch)"
    exit 1
fi
cputest_url="${cdn_success_url}https://github.com/oneclickvirt/cputest/releases/download/output/${cputest_filename}"
if ! download_file "$cputest_url" "cputest"; then
    echo "Failed to download cputest"
    exit 1
fi
chmod 777 cputest
cp cputest /usr/bin/cputest
