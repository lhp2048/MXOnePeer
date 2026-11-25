package adminboard

func pageHTML(pill, state, host, ips, keyState, notice, loginError, token, requested, login, mcp string) string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>MXOnePeer</title>
<style>
:root { color-scheme: light; }
body { margin: 0; min-height: 100vh; background: #f3efe6; color: #1c1915; font-family: "Avenir Next", "Segoe UI", sans-serif; }
.shell { max-width: 40rem; margin: 0 auto; padding: 2.5rem 1.25rem 3rem; }
header { display: flex; align-items: center; gap: 0.9rem; }
.mark { width: 2.7rem; height: 2.7rem; margin: 0; border-radius: 0.8rem; background: #0f6e6e; color: #f7f3ea; display: grid; place-items: center; font-weight: 700; letter-spacing: 0.04em; }
h1 { margin: 0; font-size: 1.45rem; letter-spacing: -0.03em; }
.sub { margin: 0.15rem 0 0; color: #625c53; font-size: 0.92rem; }
.pill { margin-left: auto; border-radius: 999px; padding: 0.28rem 0.7rem; font-size: 0.82rem; font-weight: 650; }
.on { background: #e5f4ea; color: #146c43; }
.off { background: #f3e6e1; color: #8a3b2d; }
.facts { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.7rem; margin: 1.3rem 0; }
.fact { background: #fffdf8; border: 1px solid #e4ddd0; border-radius: 0.9rem; padding: 0.8rem 0.9rem; }
.fact span { display: block; color: #625c53; font-size: 0.78rem; }
.fact strong { display: block; margin-top: 0.25rem; font-size: 0.95rem; overflow-wrap: anywhere; }
.card { background: #fffdf8; border: 1px solid #e4ddd0; border-radius: 1rem; padding: 1.1rem 1.15rem 1.2rem; margin-top: 0.8rem; }
h2 { margin: 0 0 0.35rem; font-size: 1.05rem; }
.card p { margin: 0.35rem 0 0; color: #4e4942; }
label { display: block; margin-top: 0.9rem; font-size: 0.86rem; color: #4e4942; }
input[type=password], input[type=text] { width: 100%; box-sizing: border-box; margin-top: 0.35rem; padding: 0.7rem 0.75rem; border-radius: 0.7rem; border: 1px solid #d9d0c2; background: #fff; color: #1c1915; font: inherit; }
button { margin-top: 0.9rem; border: 0; border-radius: 0.7rem; padding: 0.65rem 0.95rem; font: inherit; font-weight: 650; cursor: pointer; }
.primary { background: #0f6e6e; color: #f7f3ea; }
.quiet { background: #efeae1; color: #1c1915; }
.banner { margin-top: 0.8rem; padding: 0.65rem 0.75rem; border-radius: 0.7rem; background: #e7f3f3; color: #0f4f4f; }
.bad { background: #f8ebe6; color: #8a3b2d; }
.link { display: inline-block; margin-top: 0.8rem; color: #0f6e6e; font-weight: 700; }
.url, .mcp { overflow-wrap: anywhere; font-family: ui-monospace, "SFMono-Regular", monospace; font-size: 0.86rem; }
footer { margin-top: 1.1rem; color: #625c53; font-size: 0.86rem; }
.hint { color: #625c53; }
@media (max-width: 640px) { .facts { grid-template-columns: 1fr; } header { align-items: flex-start; } .pill { margin-top: 0.2rem; } }
</style>
</head>
<body>
<main class="shell">
<header>
<p class="mark">MX</p>
<div><h1>MXOnePeer</h1><p class="sub">本机登录控制</p></div>
<span class="pill ` + pill + `">` + state + `</span>
</header>
<section class="facts">
<div class="fact"><span>主机名</span><strong>` + host + `</strong></div>
<div class="fact"><span>Tailscale 地址</span><strong>` + ips + `</strong></div>
<div class="fact"><span>auth key</span><strong>` + keyState + `</strong></div>
</section>
` + notice + loginError + `
<section class="card">
<h2>节点名称</h2>
<p>向 Tailscale 申请这个名称。账号里已有同名节点时，实际名称可能带序号。</p>
<form method="post" action="/hostname">
<input type="hidden" name="token" value="` + token + `">
<label>节点名称<input type="text" name="hostname" value="` + requested + `" autocomplete="off" spellcheck="false"></label>
<button class="primary" type="submit">保存节点名称</button>
</form>
</section>
<section class="card">
<h2>auth key</h2>
<p>从 Tailscale 后台复制以 tskey-auth- 开头的密钥。保存后只显示末四位。</p>
<form method="post" action="/authkey">
<input type="hidden" name="token" value="` + token + `">
<label>auth key<input type="password" name="authkey" autocomplete="off"></label>
<button class="primary" type="submit">保存 auth key</button>
</form>
</section>
<section class="card">
<h2>账号登录</h2>
<p>没有密钥时，用你的 Tailscale 账号批准这台设备。</p>
<form method="post" action="/login">
<input type="hidden" name="token" value="` + token + `">
<button class="quiet" type="submit">用 Tailscale 账号登录</button>
</form>
` + login + `
</section>
<footer>本地 MCP <span class="mcp">` + mcp + `</span></footer>
</main>
</body>
</html>
`
}
