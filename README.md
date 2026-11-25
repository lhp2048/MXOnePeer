# MXOnePeer

`mxonepeer` 可以装在 Debian 13（x86_64）或 Apple 芯片的 Mac 上。它用 Python 和 `tailscale-py` 0.7.1 把 Tailscale 嵌在本进程里。你在 Tailscale 后台生成一个 auth key，写到这台机器上，节点就会出现在你的网络里。别的设备能访问它 Tailscale 地址上的状态接口。它不给这台机器建网卡。这个 Tailscale 库仍是 Alpha。

在 Debian 上安装（用 root，不编译）：

```
sudo ./deploy/install.sh
```

在 Apple 芯片的 Mac 上安装（用当前用户，不需要 root，需要 Python 3.12 或更新）：

```
./deploy/install-macos.sh
```

Mac 会把程序装进 `~/Library/Application Support/mxonepeer`，并用登录项 `com.mxonepeer` 保持运行。Intel Mac 没有可用的预编译包。

服务起来后，在这台机器上打开 `http://127.0.0.1:8788/`。页面可以设置节点名称、粘贴 auth key，也可以开始 Tailscale 账号登录。同一个地址提供本机 MCP：`http://127.0.0.1:8788/mcp`。工具有 `status`、`set_hostname`、`set_auth_key`、`start_login`。节点名称用小写字母、数字和连字符。保存后会重新连接，页面上的当前名称以 Tailscale 返回的为准。页面和 MCP 都只听本机，不出现在局域网和 Tailscale 上。从另一台电脑操作时，先做隧道：`ssh -L 8788:127.0.0.1:8788 <这台机器>`，再在那边打开同一个地址。

也可以不打开页面。Debian 上把 auth key 写进 `/etc/mxonepeer.env`，一行，形如 `TS_AUTHKEY=tskey-auth-...`。文件权限保持 `0640`。写好后执行 `sudo systemctl restart mxonepeer`。Mac 上写进 `~/Library/Application Support/mxonepeer/env`，权限 `0600`，然后执行 `launchctl kickstart -k gui/$(id -u)/com.mxonepeer`。

在另一台已登录同一账号的设备上检查：

```
curl http://mxonepeer:18020/status
```
