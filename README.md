<h1 align="center">SubMill</h1>

<p align="center">
  <a href="https://github.com/rebecaachambers/submill"><img src="https://img.shields.io/github/v/release/rebecaachambers/submill?style=flat-square&include_prereleases&label=version" /></a>
  <a href="https://github.com/rebecaachambers/submill"><img src="https://img.shields.io/github/downloads/rebecaachambers/submill/total.svg?style=flat-square" /></a>
  <a href="https://github.com/rebecaachambers/submill/issues"><img src="https://img.shields.io/github/issues-raw/rebecaachambers/submill.svg?style=flat-square&label=issues" /></a>
  <a href="https://github.com/rebecaachambers/submill/blob/master/LICENSE"><img src="https://img.shields.io/github/license/rebecaachambers/submill?style=flat-square" /></a>
</p>

---

> **SubMill** = 代理节点全自动检测 + Mihomo 代理核心，离线安装，零 HTTP 内部通信。

> 完整配置说明请参考 [config.example.yaml](config/config.example.yaml)。
> 配置变更历史请参考 [config.example.yaml 的提交记录](https://github.com/rebecaachambers/submill/commits/master/config/config.example.yaml)。

---

## 功能特性

- **全自动节点检测**：存活检测、流媒体解锁、速度测试一气呵成
- **Mihomo 内核集成**：文件直读模式，无需 HTTP/端口通信
- **配置文件分离**：SubMill 用 `submill.yaml`，Mihomo 用 `config.yaml`，互不覆盖
- **智能测速排序**：自动筛选高速节点并按速度降序排列
- **离线一键安装**：所有依赖已打包，`bash scripts/setup.sh` 即可
- **自动更新订阅**：定时拉取远程订阅源，合并去重后检测
- **节点均衡负载**：Mihomo 内置 url-test 自动优选 + load-balance 均衡分流
- **Web 控制面板**：浏览器管理配置、查看状态和日志
- **Sub-Store 集成**：内置 Sub-Store，支持多格式订阅转换
- **消息推送通知**：支持 Apprise 多渠道通知检测结果

---


## 系统要求

| 项目 | 最低配置 | 推荐配置 |
|---|---|---|
| **操作系统** | Linux (Debian/Ubuntu/CentOS/Alpine) | Debian 12+ / Ubuntu 22.04+ |
| **架构** | AMD64 / ARM64 | ARM64 (树莓派、软路由) 或 AMD64 |
| **内存** | 512MB | 1GB+ |
| **磁盘** | 500MB | 2GB+ (含 Go 编译缓存) |
| **网络** | 能访问 GitHub 拉取订阅源 | 带宽 ≥ 10Mbps |
| **CPU** | 单核 | 双核+ (并发检测 200+ 节点时) |

### 注意事项

- **ARM64 设备**（树莓派、Rockchip 软路由等）实测可用，安装脚本自动识别
- **并发数 `concurrent`**：512MB 内存建议 ≤ 100，1GB 内存可设 200~300
- **磁盘空间**：项目源码约 50MB，编译后二进制约 30MB，Go 缓存约 200MB
- **完全离线安装**：项目自带 Go 安装包 (assets/go/) 和 vendor 依赖，无需网络下载编译依赖

## 快速安装

### Linux 一键安装（推荐）

```bash
# 克隆项目
git clone https://github.com/rebecaachambers/submill.git
cd submill

# 一键安装（编译 + 配置 + 注册 systemd 服务）
bash scripts/setup.sh
```

### 管理命令

```bash
systemctl start submill    # 启动
systemctl stop submill     # 停止
systemctl restart submill  # 重启
systemctl status submill   # 状态
journalctl -u submill -f   # 实时日志
```

---

## 使用方法

### 代理端口

SubMill 启动后可通过 Mihomo 代理上网：

```bash
# HTTP 代理
curl -x http://127.0.0.1:7890 https://www.google.com

# SOCKS5 代理
curl --socks5 127.0.0.1:7890 https://www.google.com

# 环境变量
export HTTP_PROXY=http://127.0.0.1:7890
export HTTPS_PROXY=http://127.0.0.1:7890
```

> 如果节点来自 GitHub 等需要代理的地址，可配置 `github-proxy` 加速拉取：
> ```yaml
> github-proxy: "https://ghfast.top/"
> ```

### 限速配置

```yaml
# 100MB 下载限制
download-mb: 100

# 1GB 下载限制
download-mb: 1024

# 不限速
download-mb: 0
```

---

## 端口说明

| 服务 | 端口 | 用途 |
|---|---|---|
| submill | `8199` | Web 控制面板 (`/admin`)，订阅文件服务 (`/sub/`) |
| submill | `8299` | Sub-Store 管理界面 |
| mihomo | `7890` | HTTP/SOCKS5 代理端口 |

---

## SubMill + Mihomo 工作原理

```
SubMill 检测完节点后写入 config/output/all.yaml
Mihomo 通过文件类型 (type: file) proxy-provider 直接读取，无需 HTTP 传输
```

Mihomo 配置文件 (`config/config.yaml`) 中的 proxy-provider 配置：

```yaml
proxy-providers:
  submill:
    type: file
    path: output/all.yaml
    health-check:
      enable: true
      url: http://www.gstatic.com/generate_204
      interval: 300
```

---

## ⚙️ 推荐配置

以下是经过验证的推荐配置，直接复制使用即可保证最佳效果。**两个配置文件已分离，请勿混淆。**

### SubMill 配置 (`config/submill.yaml`)

```yaml
# ============ 基础设置 ============
print-progress: true
concurrent: 200                    # 并发检测数，根据机器性能调整 (100~500)
check-interval: 120                # 检测间隔(分钟)，即每2小时更新一次节点
listen-port: ":8199"               # Web 面板端口
save-method: "local"               # 保存方式：local 本地存储，无需外部依赖

# ============ 超时 & 测速 ============
timeout: 5000                      # 节点延迟超时(毫秒)
alive-test-url: http://www.gstatic.com/generate_204
speed-test-url: https://github.com/AaronFeng753/Waifu2x-Extension-GUI/releases/download/v2.21.12/Waifu2x-Extension-GUI-Portable.7z
min-speed: 512                     # 最低速度(KB/s)，低于此值丢弃
download-timeout: 10               # 测速最长等待(秒)
download-mb: 20                    # 单节点测速最大下载量(MB)

# ============ 订阅源 ============
sub-urls-retry: 3                  # 订阅拉取失败重试次数
sub-urls-concurrent: 20            # 同时拉取订阅的并发数
sub-urls-get-ua: "clash.meta"

# 远程订阅清单（自动合并多个来源）
sub-urls-remote:
  - https://raw.githubusercontent.com/beck-8/sub-urls/refs/heads/main/%E5%B0%8F%E8%80%8C%E7%BE%8E.txt

# 本地订阅地址（可直接添加自己的机场订阅）
sub-urls:
  # - https://your-subscription-url-here

# ============ 输出 & 历史 ============
keep-days: 0                       # 保留历史节点天数，0=关闭
rename-node: true                  # 节点重命名（国家+速度标签）
node-prefix: ""                    # 节点名前缀

# ============ 流媒体检测 (可选) ============
media-check: false                 # 开启后会消耗更多时间，按需启用
media-check-timeout: 5
platforms:
  - iprisk
  - youtube
  - netflix
  - openai
```

### Mihomo 配置 (`config/config.yaml`)

以下为 **自动生成** 的配置（由 SubMill 启动时调用 `WriteMihomoConfig()` 写入），仅作参考：

```yaml
mixed-port: 7890                   # HTTP/SOCKS5 混合端口
bind-address: "*"                  # 监听所有网卡，允许局域网使用
allow-lan: true
mode: rule
log-level: info
ipv6: false

# 关闭 GeoIP 自动更新（离线环境必需）
geo-auto-update: false
geo-update-interval: 99999

# DNS 使用阿里公共 DNS（无需 GeoIP MMDB）
dns:
  enable: true
  ipv6: false
  enhanced-mode: fake-ip
  nameserver:
    - 223.5.5.5
    - 119.29.29.29

# 文件类型 provider — 直接读取 SubMill 输出，无需 HTTP
proxy-providers:
  submill:
    type: file
    path: output/all.yaml
    health-check:
      enable: true
      url: http://www.gstatic.com/generate_204
      interval: 300

# 代理组：auto=自动测速优选，balance=均衡负载
proxy-groups:
  - name: PROXY
    type: select
    proxies: [auto, balance, DIRECT]

  - name: auto
    type: url-test
    use: [submill]
    url: http://www.gstatic.com/generate_204
    interval: 300
    tolerance: 20

  - name: balance
    type: load-balance
    use: [submill]
    url: http://www.gstatic.com/generate_204
    interval: 300
    strategy: consistent-hashing

# 规则：内网直连，其余走代理
rules:
  - IP-CIDR,192.168.0.0/16,DIRECT
  - IP-CIDR,10.0.0.0/8,DIRECT
  - IP-CIDR,172.16.0.0/12,DIRECT
  - IP-CIDR,127.0.0.0/8,DIRECT
  - MATCH,PROXY
```

### 关键注意事项

- **两个配置文件已分离**：SubMill 用 `config/submill.yaml`，Mihomo 用 `config/config.yaml`，不会互相覆盖
- **无需设置 Mihomo API**：Mihomo 通过 `type: file` 直接读取 `output/all.yaml`，不依赖 HTTP/端口通信
- **无需手动填写订阅链接到 Mihomo 配置**：SubMill 自动生成节点文件，Mihomo 自动读取
- **`concurrent` 不要设太高**：家用宽带建议 200~300，VPS 可到 500+，过高会导致测速不准
- **离线安装**：所有依赖 (Go、vendor) 已包含在项目中，`bash scripts/setup.sh` 一键安装

---

## 致谢

本项目基于以下优秀开源项目：

- **[subs-check](https://github.com/beck-8/subs-check)** — 感谢 [@beck-8](https://github.com/beck-8) 开发的订阅节点检测与转换工具
- **[Mihomo](https://github.com/MetaCubeX/mihomo)** — 感谢 [MetaCubeX](https://github.com/MetaCubeX) 团队开发的强大代理核心
