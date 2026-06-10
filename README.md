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
| **架构** | AMD64 / ARM64 | ARM64 或 AMD64 |
| **内存** | 2GB + 1GB swap | 4GB+ |
| **磁盘** | 2GB | 4GB+ (含 Go 编译缓存 ~500MB) |
| **网络** | 能访问 GitHub 拉取订阅源 | 带宽 ≥ 10Mbps |
| **CPU** | 单核 | 双核+ |

### 实机验证

| 设备 | 内存 | 结果 |
|---|---|---|
| LXC 容器 (114) | 512MB | ❌ Go 编译 OOM 被 kill，无法安装 |
| 软路由 ARM64   | 3.6GB | ✅ 正常运行，217 节点并发 200 稳定 |

### 注意事项

- **编译阶段内存需求大**：`go build` 链接 mihomo 内核需 1~1.5GB，小于 2GB 必须开 swap
- **运行时内存**：SubMill ~150MB + Mihomo ~50MB + SubStore ~100MB，合计约 300MB
- **并发数建议**：2GB 内存 ≤ 100，4GB 可设 200~300，8GB 可到 500
- **ARM64 设备**：安装脚本自动识别架构，实测树莓派/软路由可用
- **磁盘空间**：项目源码约 50MB，编译产物约 30MB，Go 编译缓存 300~500MB
- **必须开启 swap**：低于 4GB 内存务必配置至少 2GB swap，否则编译阶段大概率被 OOM kill

```bash
# 检查 swap
free -h

# 如果没有 swap，创建 4-8GB swap 文件
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

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


### Docker 安装

```bash
# 构建镜像
docker build -t submill .

# 运行容器
docker run -d --name submill --restart=always \
  -p 7890:7890 \
  -p 8199:8199 \
  -v submill-config:/app/config \
  -v submill-output:/app/output \
  submill

# 查看日志
docker logs -f submill

# 停止/启动
docker stop submill
docker start submill
```

### Docker Compose 安装（推荐）

```bash
# 拉取项目
git clone https://github.com/rebecaachambers/submill.git
cd submill

# 构建并启动
docker compose up -d

# 查看日志
docker compose logs -f

# 停止
docker compose down
```


| 参数 | 说明 |
|---|---|
| `-p 7890:7890` | Mihomo HTTP/SOCKS5 代理端口 |
| `-p 8199:8199` | SubMill Web 面板 |
| `-v submill-config:/app/config` | 持久化配置文件 (`submill.yaml` + `config.yaml`) |
| `-v submill-output:/app/output` | 持久化节点输出文件 (`all.yaml`) |

> **注意**：首次运行后编辑 `submill-config` volume 中的 `submill.yaml`，填入 `sub-urls` 订阅地址。

---

## 工作流程

```
SubMill (subs-check)        Worker (watch-submill)        Mihomo
───────────────             ──────────────                ──────
config/submill.yaml    ->   inotifywait 监听   ->        mihomo/nodes/all.yaml
output/all.yaml              检测变化->转换->写入           (读取节点)
(输出节点)

3 个 systemd 服务，顺序启动:
  submill (8199) -> watch-submill -> mihomo (7890)
```

| 服务 | 职责 |
|---|---|
| **SubMill** | 拉取订阅、检测节点（存活/流媒体/速度），输出到 `output/all.yaml` |
| **Worker** | 监听 `output/all.yaml` 变化，转换后写入 `mihomo/nodes/all.yaml`，通知 Mihomo 重载 |
| **Mihomo** | 读取 `mihomo/nodes/all.yaml`，提供 HTTP/SOCKS5 代理（7890），自动测速优选 + 均衡负载 |

## 端口说明

| 服务 | 端口 | 用途 |
|---|---|---|
| submill | `8199` | Web 控制面板 (`/admin`)，订阅文件服务 (`/sub/`) |
| submill | `8299` | Sub-Store 管理界面 |
| mihomo | `7890` | HTTP/SOCKS5 代理端口 |

---

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
