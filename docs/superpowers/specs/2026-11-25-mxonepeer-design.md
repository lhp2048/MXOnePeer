# MXOnePeer 设计

日期：2026-11-25

MXOnePeer 是一个 Python 程序，可以装在 Debian 13（x86_64，系统 Python 3.13）或 Apple 芯片的 Mac（Python 3.12 或更新）上。程序用 `tailscale-py` 0.7.1 把 Tailscale 嵌进本进程。操作者把 Tailscale 后台生成的 auth key 放在这台机器上，进程用它登录。同一账号下的其他设备能看到这个节点，并能访问本进程在 Tailscale 地址上提供的状态接口。

这一版只证明登录和连通。远程控制以后再加。

`tailscale-py` 0.7.1 是 Tailscale 官方的 Alpha 库，底层是尚未做完第三方安全审计的 Rust 客户端。这一版接受这个限制。目标机器上不编译：安装时用 pip 下载 PyPI 上的预编译 wheel。

## 范围

做这些：

- 一个常驻进程，节点名默认 `mxonepeer`。
- 用户态 Tailscale。不创建系统网卡，不需要 root。
- 用 auth key 或本机页面上的 Tailscale 账号登录。auth key 以 `tskey-auth-` 开头。
- 登录后的节点密钥记在 `/var/lib/mxonepeer/node-keys.json`。重启后仍是同一个节点。
- 在该节点的 Tailscale IPv4 和 IPv6 的 `8787` 端口提供 `GET /status`。

不做这些：

- 不把整机加入 Tailscale。SSH 和其他程序不能通过这个节点访问。
- 不在局域网网卡上监听。
- 不提供登出命令。管理只在本机 `127.0.0.1:8788`。
- 不实现远程控制。
- 不在目标机器上安装 Go、Rust 或编译器。

## 进程

用 Debian 的 `python3` 跑 `/opt/mxonepeer/venv` 里的解释器，入口是 `python -m mxonepeer`。

`tailscale.connect` 使用：

- 第一个参数：`/var/lib/mxonepeer/node-keys.json`。
- `auth_key`：环境变量 `TS_AUTHKEY`。
- `hostname`：启动参数 `--hostname`，默认 `mxonepeer`。

端口是 `--port`，默认 `8787`。状态目录是 `--state-dir`，默认 `/var/lib/mxonepeer`。

不读其他 Tailscale 环境变量。进程启动时清掉 `TS_AUTH_KEY`、`TS_CLIENT_SECRET`、`TS_CLIENT_ID`、`TS_ID_TOKEN`、`TS_AUDIENCE`、`TS_CONTROL_URL`、`TS_HOSTNAME`。`TS_AUTHKEY` 先读入内存再从环境里清掉。日志里出现密钥时替换成 `tskey-auth-***`。

systemd 设置 `TS_RS_EXPERIMENT=this_is_unstable_software`。这是该 Alpha 库要求的开关。

## 登录

没有密钥时进程不退出。本机后台听 `127.0.0.1:8788`，只接受来自这台机器的连接。页面有两件事：粘贴 `tskey-auth-` 开头的 auth key，或开始 Tailscale 账号登录。

auth key 写到 `<state-dir>/authkey`，权限 `0600`。环境变量 `TS_AUTHKEY` 在启动时如果有效，会写入同一个文件。页面不回显完整密钥，只显示末四位。提交必须带上页面里的一次性 token。

账号登录调用 `tailscale.connect` 且不带密钥。库若提供 `is_authorized`，页面显示其中的登录链接；批准后开始在 Tailscale 地址上提供 `/status`。库没有这个方法时，页面说明要改用 auth key。

页面也可以设置节点名称。名称写成状态目录里的 `hostname` 文件，权限 `0600`。规则是小写字母、数字和连字符，长度 1 到 63，不能用连字符开头或结尾。保存一个新名称后重新连接，并把这个名称传给 `tailscale.connect`。页面上的当前名称来自 `self_node().hostname`。

同一个本机端口提供 MCP，路径是 `POST /mcp`。协议版本接受 `2024-11-05`、`2025-03-26`、`2025-06-18`。工具有四个：`status` 查看在线状态、当前名称、要申请的名称、地址、密钥是否已配置和登录链接；`set_hostname` 保存节点名称；`set_auth_key` 保存 auth key；`start_login` 开始账号登录。返回里不包含完整密钥。请求必须是 `application/json`。

换账号：在页面或 MCP 保存新的 auth key，或停掉服务后删掉 `node-keys.json` 和 `authkey` 再登录。密钥不写入日志；异常文本里的密钥替换成 `tskey-auth-***`。

## 状态接口

只实现 `GET /status`。监听绑在 `ipv4_addr()` 和 `ipv6_addr()` 返回的地址上，端口为 `--port`。不绑 `0.0.0.0`，也不绑局域网地址。

在 `tailscale-py` 的 TCP 流上读写 HTTP/1.1，一条连接处理一个请求，然后关闭。请求头超过 5 秒或超过 8192 字节，关闭连接。

- 其他路径返回 `404`。
- 其他方法返回 `405`，并带 `Allow: GET`。
- 成功返回 `200` 和 `Content-Type: application/json`。
- 程序不做应用层鉴权。能否从别的设备访问，由该账号的 Tailscale ACL 决定。

每次请求重新读当前地址和 `self_node()`，不缓存旧地址。正文：

```json
{
  "hostname": "mxonepeer",
  "dnsName": "mxonepeer.example.ts.net",
  "tailscaleIPs": ["100.64.0.1", "fd7a:115c:a1e0::1"],
  "online": true,
  "backendState": "Running"
}
```

字段：

- `hostname`：`self_node().hostname`。账号里已有同名节点时，这里是库报告的实际名字。
- `dnsName`：`hostname` 和 `self_node().tailnet` 用点连起来。没有 tailnet 时为空字符串。有结尾的点时去掉这一个点。
- `tailscaleIPs`：当前 IPv4 和 IPv6，IPv4 在前。无效地址丢掉。
- `online`：`backendState` 为 `Running` 且 `tailscaleIPs` 非空时为 true。
- `backendState`：至少读到一个 Tailscale 地址时为 `Running`，否则为 `Offline`。

端口能接受连接并返回 `200`，即表示这个进程在该 Tailscale 网络里并且正在服务。

## 失败与停止

- 状态目录不能创建或不能写：向标准错误写出原因，退出码 `1`。
- `tailscale.connect` 失败：写出去掉密钥后的原因，退出码 `1`。
- 控制面暂时不可达时，由库自己重试的部分留在 `connect` 里。`connect` 抛错才退出。
- 单个连接读写失败：关掉该连接，进程继续。
- 收到 `SIGTERM` 或 `SIGINT`：停止接受新连接，退出码 `0`。

systemd 使用 `Restart=on-failure`。正常停止不会被立刻拉起。

## 安装

在目标机器上用 root 执行 `deploy/install.sh`。脚本：

1. 要求 `python3` 至少 3.13。创建虚拟环境失败时，提示安装 `python3-venv`。
2. 创建系统用户和组 `mxonepeer`：`--system`，shell 为 `/usr/sbin/nologin`，家目录为 `/var/lib/mxonepeer`。已存在则不修改。
3. 创建 `/var/lib/mxonepeer`，所有者是 `mxonepeer`，权限 `0750`。
4. 把 `mxonepeer` 包拷到 `/opt/mxonepeer/mxonepeer`。
5. 用 `python3 -m venv /opt/mxonepeer/venv`，再执行 `/opt/mxonepeer/venv/bin/pip install tailscale-py==0.7.1`。这一步下载预编译 wheel，不在机器上编译。
6. 安装 systemd 单元。
7. `/etc/mxonepeer.env` 不存在时，创建一个只有 `TS_AUTHKEY=` 的文件，所有者 root，组 `mxonepeer`，权限 `0640`。已有文件不覆盖。
8. `enable` 并 `restart`。没有密钥时服务也保持运行，以便打开本机登录页。

服务：

- `User=mxonepeer`，`Group=mxonepeer`
- `Environment=PYTHONPATH=/opt/mxonepeer`
- `Environment=TS_RS_EXPERIMENT=this_is_unstable_software`
- `EnvironmentFile=/etc/mxonepeer.env`
- `ExecStart=/opt/mxonepeer/venv/bin/python -m mxonepeer --hostname mxonepeer --state-dir /var/lib/mxonepeer --port 8787`
- `After=network-online.target`，`Wants=network-online.target`
- `Restart=on-failure`，`RestartSec=5`
- `NoNewPrivileges=true`

在 Apple 芯片的 Mac 上，由当前用户执行 `deploy/install-macos.sh`，不使用 root。脚本要求 Python 至少 3.12。程序和虚拟环境放在 `~/Library/Application Support/mxonepeer`，状态文件也在这个目录。`~/Library/LaunchAgents/com.mxonepeer.plist` 在登录后启动它，异常退出后重新拉起。可选的 auth key 写在同目录的 `env` 文件，权限 `0600`，格式与 Debian 的环境文件相同。Intel Mac 没有预编译 wheel，脚本直接退出。

密钥不写进仓库，也不写进安装脚本。

## 代码布局

```
MXOnePeer/
  requirements.txt
  mxonepeer/__init__.py
  mxonepeer/__main__.py
  mxonepeer/main.py
  mxonepeer/runtime.py
  mxonepeer/admin.py
  mxonepeer/mcp.py
  mxonepeer/hostname.py
  mxonepeer/keys.py
  mxonepeer/status.py
  mxonepeer/httpio.py
  mxonepeer/node.py
  tests/test_status.py
  tests/test_httpio.py
  tests/test_admin.py
  tests/test_main.py
  deploy/mxonepeer.service
  deploy/install.sh
  deploy/install-macos.sh
  README.md
```

职责：

- `status`：把主机名、DNS 名、地址和后端状态格式化成 JSON。不导入 tailscale。
- `httpio`：解析一个 HTTP 请求，生成 `200`、`404` 或 `405`。不导入 tailscale。
- `hostname`：检查和保存节点名称。
- `keys`：检查、保存和掩码 auth key。
- `admin`：本机登录页。
- `mcp`：本机 MCP，工具是 `status`、`set_hostname`、`set_auth_key`、`start_login`。
- `node`：调用 `tailscale-py`，在 Tailscale 地址上接受连接并交出状态。
- `runtime`：按密钥或账号登录维持连接。
- `main`：解析参数、接信号。

`requirements.txt` 只有一行：`tailscale-py==0.7.1`。

## 验证

自动测试不连接 Tailscale，也不安装 `tailscale-py`：

- `Running` 且同时有 IPv6 和 IPv4 时，`online` 为 true，IPv4 在前，`hostname` 用传入的实际名字。结尾的一个点从 `dnsName` 去掉。
- `Running` 但没有地址时，`online` 为 false，`tailscaleIPs` 为 `[]`。
- 非 `Running` 时，`online` 为 false。
- `dnsName` 为空时，JSON 里仍有这个字段，值为 `""`。
- `GET /status` 得到 `200` 和 `application/json`。
- `POST /status` 得到 `405` 和 `Allow: GET`。
- `GET /other` 得到 `404`。

命令是 `python3 -m unittest discover -s tests`。

装到目标机器后的人工确认：

1. `/etc/mxonepeer.env` 里写好 auth key，服务处于运行。
2. Tailscale 管理页或其他已登录设备的 `tailscale status` 里能看到这个节点。
3. 在另一台已加入同一账号的设备上执行 `curl http://mxonepeer:8787/status`。返回 `200`，`online` 为 true，`tailscaleIPs` 里有 `100.` 开头的地址。MagicDNS 不可用时，改用 Tailscale IP 和端口 `8787`。
4. 目标机器的系统网卡上没有 Tailscale 接口。对 `127.0.0.1:8787` 和局域网地址的 `8787` 连接失败。

## 完成标准

目标机器上的 `mxonepeer` 服务用操作者的 auth key 保持在线。其他设备能看到该节点，并能从 Tailscale 地址取到 `/status`。安装过程没有在目标机器上编译。目标机器的系统网卡上没有 Tailscale 接口。
