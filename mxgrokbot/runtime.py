import asyncio
import os
import secrets
import sys

from mxgrokbot.admin import Board, authorized, extract_login_url, serve_admin
from mxgrokbot.keys import normalize_auth_key, read_auth_key, write_auth_key
from mxgrokbot.node import connect_device, serve_device


async def run(hostname, state_dir, port, admin_host, admin_port, env_key, stop):
    prepare_state_dir(state_dir)
    if env_key:
        try:
            write_auth_key(state_dir, normalize_auth_key(env_key))
        except ValueError as exc:
            print(str(exc), file=sys.stderr)
    board = Board(secrets.token_urlsafe(16))
    board.refresh_key(state_dir)
    server = await serve_admin(admin_host, admin_port, board, state_dir)
    print("local admin http://%s:%d/" % (admin_host, admin_port), file=sys.stderr)
    try:
        while not stop.is_set():
            action = board.action
            board.action = ""
            if action == "login":
                await _interactive(hostname, state_dir, port, board, stop)
                continue
            key = read_auth_key(state_dir)
            if key:
                await _keyed(key, hostname, state_dir, port, board, stop)
                continue
            await _wait(board, stop)
    finally:
        server.close()
        await server.wait_closed()


async def _keyed(key, hostname, state_dir, port, board, stop):
    started = board.generation
    session_stop = asyncio.Event()
    task = asyncio.create_task(_keyed_session(key, hostname, state_dir, port, board, session_stop))
    await _until_change(task, started, board, stop)
    session_stop.set()
    await _finish(task)


async def _keyed_session(key, hostname, state_dir, port, board, session_stop):
    dev = await connect_device(key, hostname, state_dir)
    board.login_url = ""
    board.login_error = ""
    board.notice = "auth key 已连接。"
    await serve_device(dev, port, session_stop, board)


async def _interactive(hostname, state_dir, port, board, stop):
    started = board.generation
    session_stop = asyncio.Event()
    task = asyncio.create_task(_interactive_session(hostname, state_dir, port, board, session_stop, stop))
    await _until_change(task, started, board, stop)
    session_stop.set()
    await _finish(task)


async def _interactive_session(hostname, state_dir, port, board, session_stop, stop):
    dev = await connect_device("", hostname, state_dir)
    checker = getattr(dev, "is_authorized", None)
    if checker is None:
        board.login_url = ""
        board.login_error = "当前 tailscale-py 不能返回账号登录链接，请填写 auth key。"
        return
    while not session_stop.is_set() and not stop.is_set():
        state = await checker()
        url = extract_login_url(state)
        if url:
            board.login_url = url
            board.notice = "用浏览器打开登录链接，使用你的 Tailscale 账号批准这台设备。"
        if authorized(state):
            board.login_url = ""
            board.login_error = ""
            board.notice = "账号已登录。"
            await serve_device(dev, port, session_stop, board)
            return
        await asyncio.sleep(2)


async def _until_change(task, started, board, stop):
    while not stop.is_set() and board.generation == started and not task.done():
        wake = asyncio.create_task(board.wake.wait())
        halted = asyncio.create_task(stop.wait())
        done, pending = await asyncio.wait(
            {task, wake, halted},
            return_when=asyncio.FIRST_COMPLETED,
        )
        for item in pending:
            item.cancel()
        for item in pending:
            try:
                await item
            except (asyncio.CancelledError, Exception):
                pass
        if task in done:
            error = task.exception()
            if error is not None:
                board.online = False
                board.login_error = _public_error(error)
                print("tailscale: %s" % board.login_error, file=sys.stderr)
                await asyncio.sleep(3)
            return
        if board.wake.is_set():
            board.wake.clear()


async def _wait(board, stop):
    board.wake.clear()
    wake = asyncio.create_task(board.wake.wait())
    halted = asyncio.create_task(stop.wait())
    _done, pending = await asyncio.wait({wake, halted}, return_when=asyncio.FIRST_COMPLETED)
    for item in pending:
        item.cancel()
    for item in pending:
        try:
            await item
        except (asyncio.CancelledError, Exception):
            pass
    if board.wake.is_set():
        board.wake.clear()


async def _finish(task):
    if not task.done():
        task.cancel()
    try:
        await task
    except (asyncio.CancelledError, Exception):
        return


def _public_error(error):
    text = str(error)
    marker = "tskey-auth-"
    while marker in text:
        start = text.find(marker)
        end = start + len(marker)
        while end < len(text) and text[end] not in " \n\r\t":
            end += 1
        text = text[:start] + "tskey-auth-***" + text[end:]
    return text


def prepare_state_dir(state_dir):
    os.makedirs(state_dir, mode=0o750, exist_ok=True)
    os.chmod(state_dir, 0o750)
