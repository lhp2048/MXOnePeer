# MXOnePeer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run a Python process on Debian 13 that joins the operator's tailnet with `tailscale-py` and serves `GET /status` on that node's Tailscale addresses.

**Architecture:** `mxonepeer.status` and `mxonepeer.httpio` are pure and tested. `mxonepeer.node` is the only module that imports `tailscale`. systemd runs `/opt/mxonepeer/venv/bin/python -m mxonepeer` with `TS_AUTHKEY` from `/etc/mxonepeer.env`.

**Tech Stack:** Python 3.13, `tailscale-py==0.7.1` (prebuilt manylinux x86_64 wheel), systemd.

## Global Constraints

- GrokBot is Debian 13, x86_64, system Python 3.13. Do not compile on that machine. Do not install Go or Rust there.
- Embed Tailscale with `tailscale.connect(state_file, auth_key, hostname=...)`. Userspace only. No TUN and no root for the service process.
- Login is `TS_AUTHKEY` and the value must start with `tskey-auth-`. There is no browser login in this version.
- State file is `/var/lib/mxonepeer/node-keys.json`. Directory mode is `0750`.
- Listen with `tcp_listen` on `ipv4_addr()` and `ipv6_addr()`, port `8787`. Do not bind a LAN address.
- `GET /status` returns the JSON document in the spec. Other paths are `404`. Other methods are `405` with `Allow: GET`. One request per connection.
- Pin `tailscale-py==0.7.1`. The unit sets `TS_RS_EXPERIMENT=this_is_unstable_software`.
- Never write an auth key into the repo, the unit, or `install.sh`.
- Automated tests must not import `tailscale` and must not join a tailnet.

The Python tree, unit, installer, and README in this directory implement the spec. Do not revive the earlier Go/`tsnet` plan.

---

### Task 1: Pure status and HTTP

**Files:**
- Create: `mxonepeer/status.py`
- Create: `mxonepeer/httpio.py`
- Test: `tests/test_status.py`
- Test: `tests/test_httpio.py`

- [ ] **Step 1: Run the pure tests**

Run from `MXOnePeer/`:

```bash
python3 -m unittest discover -s tests -v
```

Expected: `test_status` and `test_httpio` PASS. `test_main` PASS without opening a tailnet.

### Task 2: Node, entrypoint, install

**Files:**
- Create: `mxonepeer/node.py`
- Create: `mxonepeer/main.py`
- Create: `mxonepeer/__main__.py`
- Create: `deploy/mxonepeer.service`
- Create: `deploy/install.sh`
- Create: `requirements.txt`
- Create: `README.md`

- [ ] **Step 1: Check the installer syntax**

```bash
sh -n deploy/install.sh
```

Expected: exit 0.

- [ ] **Step 2: Confirm the service points at the venv interpreter and the env file**

`deploy/mxonepeer.service` uses `/opt/mxonepeer/venv/bin/python`, `EnvironmentFile=/etc/mxonepeer.env`, and `--port 8787`.

### Task 3: Manual check on GrokBot

Copy this directory to the Debian 13 machine and run:

```bash
sudo ./deploy/install.sh
```

Write `TS_AUTHKEY=tskey-auth-...` into `/etc/mxonepeer.env`, then:

```bash
sudo systemctl enable --now mxonepeer
journalctl -u mxonepeer
```

From another device on the same account:

```bash
curl http://mxonepeer:8787/status
```

Expected: HTTP 200, `"online": true`, and a `tailscaleIPs` value starting with `100.`. `ip addr` on GrokBot shows no Tailscale interface. Connections to `127.0.0.1:8787` and to the LAN address on port 8787 fail.
