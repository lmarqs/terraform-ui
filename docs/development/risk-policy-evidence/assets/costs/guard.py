"""Bounded supervisor reused from reviewed D-03b; configured by run.py."""
import os
import json
from pathlib import Path
import selectors
import signal
import subprocess
import time

MAX = 1048576

def guarded(name, command, build=False, wall=None, memory=None):
    seconds = wall or (120 if build else 30)
    cap = memory or (4294967296 if build else 1073741824)
    rss = OUT / f'{name}.rss.txt'
    cmd = ['/usr/bin/time', '-v', '-o', str(rss), 'prlimit', f'--as={cap}', '--core=0',
           '--', 'timeout', '--signal=TERM', '--kill-after=2s', f'{seconds}s', *command]
    started = time.monotonic()
    p = subprocess.Popen(cmd, cwd=SCRATCH, env=ENV, start_new_session=True,
                         stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    sel = selectors.DefaultSelector()
    buffers = {p.stdout: bytearray(), p.stderr: bytearray()}
    for f in buffers:
        os.set_blocking(f.fileno(), False)
        sel.register(f, selectors.EVENT_READ)
    limit = 'none'
    truncated = {'stdout': False, 'stderr': False}

    def drain_ready():
        nonlocal limit
        for key, _ in sel.select(0.05):
            part = os.read(key.fd, 65536)
            if not part:
                sel.unregister(key.fileobj)
                continue
            dest = buffers[key.fileobj]
            room = MAX - len(dest)
            dest.extend(part[:room])
            if len(part) > room:
                channel = 'stdout' if key.fileobj is p.stdout else 'stderr'
                truncated[channel] = True
                limit = 'channel_overflow'

    while sel.get_map():
        if time.monotonic() - started > seconds + 3:
            limit = 'supervisor_timeout'
            break
        drain_ready()
        if limit != 'none':
            break
    if limit == "none":
        try:
            p.wait(timeout=max(0.1, seconds + 3 - (time.monotonic() - started)))
        except subprocess.TimeoutExpired:
            limit = "supervisor_timeout"
    # Kill descendants even when the leader has already exited and closed its pipes.
    for sig in (signal.SIGTERM, signal.SIGKILL):
        try:
            os.killpg(p.pid, sig)
        except ProcessLookupError:
            break
        if sig == signal.SIGTERM:
            time.sleep(0.05)
    p.wait(timeout=3)
    drain_deadline = time.monotonic() + 2
    while sel.get_map() and time.monotonic() < drain_deadline:
        drain_ready()
    pipes_drained = not sel.get_map()
    if not pipes_drained and limit == 'none':
        limit = 'undrained_pipes'
    sel.close()
    survivors = []
    for entry in Path('/proc').iterdir():
        if not entry.name.isdigit():
            continue
        try:
            fields = (entry / 'stat').read_text().rsplit(')', 1)[1].split()
            if int(fields[2]) == p.pid and fields[0] != 'Z':
                survivors.append(int(entry.name))
        except (FileNotFoundError, ProcessLookupError, PermissionError):
            pass
    if survivors:
        limit = 'surviving_group_members'
    stdout, stderr = (bytes(buffers[f]).decode(errors='replace') for f in (p.stdout, p.stderr))
    p.stdout.close(); p.stderr.close()
    if p.returncode == 124 and limit == 'none':
        limit = 'timeout'
    elif p.returncode != 0 and limit == 'none':
        limit = 'nonzero_exit'
    record = {'command': cmd, 'exit': p.returncode, 'external_limit': limit,
              'stdout': stdout, 'stderr': stderr, 'wall_seconds': time.monotonic() - started,
              'rss_file': rss.name, 'truncated': truncated, 'pipes_drained': pipes_drained,
              'surviving_running_group_members': survivors,
              'peak_rss_kib': int(next(line.split(':',1)[1].strip() for line in rss.read_text().splitlines() if 'Maximum resident set size' in line)),
              'status': 'ok' if p.returncode == 0 and limit == 'none' else 'failed'}
    (OUT / f'{name}.process.json').write_text(json.dumps(record, indent=2) + '\n')
    return record


