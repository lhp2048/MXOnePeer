# MXOnePeer

MXOnePeer 把 Tailscale 嵌在本进程里。节点出现在你的 Tailscale 网络中，别的设备可以访问 `http://mxonepeer:18020/status`。它不给这台机器建网卡，SSH 和其他程序不能通过这个节点访问。

仓库里有两个实现，选一个安装：

- `python/` 使用 `tailscale-py` 0.7.1。可以装在 Debian 13（x86_64）、Apple 芯片的 Mac、64 位 Windows。Intel Mac 没有对应的预编译包。
- `go/` 使用官方 `tailscale.com/tsnet`。可以装在 Debian（x86_64）、Apple 芯片 Mac、Intel Mac、64 位 Windows。目标机器上有对应的预编译文件时，不需要安装 Go。

两个实现的页面、MCP 和端口相同。本机打开 `http://127.0.0.1:8788/`，可以设置节点名称、保存 auth key，或开始 Tailscale 账号登录。MCP 在 `http://127.0.0.1:8788/mcp`，工具是 `status`、`set_hostname`、`set_auth_key`、`start_login`。页面和 MCP 只听本机。从另一台电脑操作时，先做隧道：`ssh -L 8788:127.0.0.1:8788 <这台机器>`。

节点名称用小写字母、数字和连字符。保存后会重新连接，页面上的当前名称以 Tailscale 返回的为准。

## Python

```
sudo ./python/deploy/install.sh
./python/deploy/install-macos.sh
powershell -NoProfile -ExecutionPolicy Bypass -File .\python\deploy\install-windows.ps1
```

Debian 用 root。Mac 用当前用户，需要 Python 3.12 或更新，并且是 Apple 芯片。Windows 用当前用户，不要用管理员身份，需要 64 位 Python 3.12 或更新。

## Go

先在一台装了 Go 1.27 或更新版本的机器上生成四个系统的程序，文件在 `go/dist/`，不进入 git：

```
./go/deploy/build.sh
```

然后在目标机器上安装：

```
sudo ./go/deploy/install.sh
./go/deploy/install-macos.sh
powershell -NoProfile -ExecutionPolicy Bypass -File .\go\deploy\install-windows.ps1
```

Mac 安装脚本同时接受 Apple 芯片和 Intel。Intel Mac 使用 `go/dist/mxonepeer-darwin-amd64`。这个文件已经在旁边时，安装脚本不会要求本机安装 Go。

## 写入 auth key

也可以不打开页面。Debian 上把一行 `TS_AUTHKEY=tskey-auth-...` 写进 `/etc/mxonepeer.env`，权限保持 `0640`，然后执行 `sudo systemctl restart mxonepeer`。Mac 上写进 `~/Library/Application Support/mxonepeer/env`，权限 `0600`，然后执行 `launchctl kickstart -k gui/$(id -u)/com.mxonepeer`。Windows 上写进 `%LOCALAPPDATA%\mxonepeer\env`，然后执行 `Stop-ScheduledTask -TaskName MXOnePeer; Start-ScheduledTask -TaskName MXOnePeer`。

在另一台已登录同一账号的设备上检查：

```
curl http://mxonepeer:18020/status
```

## Flutter 客户端

Mac 客户端自己登录 Tailscale，并列出网络里的节点。先执行 `./go/deploy/build.sh`，再执行 `cd flutter && flutter run -d macos`。

```
./go/deploy/build.sh
cd flutter && flutter run -d macos
```

界面打开时启动 `mxonepeer-client`，本机接口是 `http://127.0.0.1:19020/`。登录状态在 `~/Library/Application Support/mxonepeer-client`。
