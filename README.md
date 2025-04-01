# cputest

[![Hits](https://hits.spiritlhl.net/cputest.svg?action=hit&title=Hits&title_bg=%23555555&count_bg=%230eecf8&edge_flat=false)](https://hits.spiritlhl.net)

[![Build and Release](https://github.com/oneclickvirt/cputest/actions/workflows/main.yaml/badge.svg)](https://github.com/oneclickvirt/cputest/actions/workflows/main.yaml)

CPU测试模块 (CPU Test Module) 

# 功能(Features)

- [x] 使用```sysbench```测试CPU得分
- [x] 使用```geekbench```测试CPU得分
- [x] 在以上两者都无法工作时，使用Golang模仿sysbench实现的程序测试进行测试，但单线程一般会比sysbench低300个左右的事件数每秒(得分)(8.8%误差)
- [x] 使用```winsat```测试CPU得分
- [x] 以```-l```指定输出的语言类型，可指定```zh```或```en```，默认不指定时使用中文输出
- [x] 以```-m```指定测试的方法，可指定```sysbench```或```geekbench```，默认不指定时使用```sysbench```进行测试
- [x] 以```-t```指定测试的线程数，可指定```single```或```multi```，默认不指定时使用单线程进行测试
- [x] 全平台编译支持
- [x] 下载```geekbench```前检测本机剩余内存是否足以进行测试，检测是否有IPV4网络以获取结果，自动切换下载的版本

## TODO

- [ ] 适配MACOS系统测试

# 环境准备

## sysbench

注意：默认不自动安装```sysbench```组件，如需使用请自行安装后再使用本项目，如

```apt update && apt install sysbench -y```

## geekbench

注意：```geekbench```测试不支持离线操作，需要确保宿主机存在IPV4网络时才能进行测试，纯IPV6网络也不能进行测试。

个人更推荐使用```sysbench```进行测试，```geekbench```测试的基准线随着版本不同是不一样的(对标版本初期最强劲的Intel的CPU)，而```sysbench```的基准线一直是5秒内算素数，不存在变动。(同等条件下```geekbench```需要测试至少2分钟)

### 检测本机内存大小以及开设虚拟内存

同等测试环境下，```sysbench```测试没有最低内存大小需求，而```geekbench```有最低内存大小需求(至少1GB内存)。

```
curl -L https://raw.githubusercontent.com/spiritLHLS/addswap/main/addswap.sh -o addswap.sh && chmod +x addswap.sh && bash addswap.sh
```

执行后若显示

```
              total        used        free      shared  buff/cache   available
Mem:            512           0         512           0           0           0
Swap:             0           0           0
```

看到```free```那一列的大小上下加起来不足```1512```时，输入数字```1```选择添加虚拟内存，然后输入```1512```增加虚拟内存。

### 下载文件

如需使用```geekbench```请事先执行

```
curl -L https://raw.githubusercontent.com/oneclickvirt/cputest/main/dgb.sh -o dgb.sh && chmod +x dgb.sh
```

然后使用```-v```指定需要后续使用的geekbench版本```gb4```或```gb5```或```gb6```

若我后续使用geekbench6进行测试则

```
bash dgb.sh -v gb6
```

下载对应版本的geekbench

# 使用(Usage)

下载、安装、升级

```
curl https://raw.githubusercontent.com/oneclickvirt/cputest/main/ct_install.sh -sSf | bash
```

或

```
curl https://cdn.spiritlhl.net/https://raw.githubusercontent.com/oneclickvirt/cputest/main/ct_install.sh -sSf | bash
```

使用

```
cputest
```

或

```
./cputest
```

进行测试

```
Usage: cputest [options]
  -h    Show help information
  -l string
        Language parameter (en or zh)
  -log
        Enable logging
  -m string
        Specific Test Method (sysbench or geekbench)
  -t string
        Specific Test Thread Mode (single or multi)
  -v    show version
```

## 卸载

```
rm -rf /root/cputest
rm -rf /usr/bin/cputest
```

## 在Golang中使用

```
go get github.com/oneclickvirt/cputest@latest
```
