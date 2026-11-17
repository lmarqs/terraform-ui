#!/usr/bin/env python3
"""Disposable Linux D-03b experiment; run from any CWD. Writes only here and /tmp."""
import hashlib
import http.server
import json
import os
from pathlib import Path
import selectors
import shutil
import signal
import subprocess
import tempfile
import threading
import time

ROOT = Path(__file__).resolve().parent
OUT = ROOT / 'output'
OUT.mkdir(exist_ok=True)
SCRATCH = Path(tempfile.mkdtemp(prefix='tfui-d03b-', dir='/tmp'))
GOROOT = '/home/lmarqs/.local/share/mise/installs/go/1.25.5'
ENV = {'PATH': f'{GOROOT}/bin:/usr/bin:/bin', 'GOROOT': GOROOT,
       'GOTOOLCHAIN': 'local', 'CGO_ENABLED': '0', 'GOMAXPROCS': '2',
       'GOPATH': '/tmp/tfui-d03b/gopath', 'GOCACHE': '/tmp/tfui-d03b/cache', 'GOMODCACHE': '/tmp/tfui-d03b/modcache',
       'LANG': 'C.UTF-8', 'TZ': 'UTC', 'D03_MARKER': 'D03_SYNTHETIC_SECRET_ONLY'}
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
              'surviving_running_group_members': survivors}
    (OUT / f'{name}.process.json').write_text(json.dumps(record, indent=2) + '\n')
    return record


# Harmless limit preflight, before OPA. Descendant PID must have exited or be a zombie.
mem = guarded('preflight-memory', ['/usr/bin/python3', '-c',
    'try:\n x=bytearray(512*1024*1024)\nexcept MemoryError:\n print("MemoryError"); raise SystemExit(42)'], memory=134217728)
assert mem['exit'] == 42 and 'MemoryError' in mem['stdout']
pidfile = SCRATCH / 'descendant.pid'
script = ('import subprocess,time,pathlib; p=subprocess.Popen(["/usr/bin/python3","-c","import time; time.sleep(20)"]); '
          f'pathlib.Path({str(pidfile)!r}).write_text(str(p.pid)); time.sleep(20)')
timed = guarded('preflight-timeout', ['/usr/bin/python3', '-c', script], wall=1)
assert timed['exit'] == 124
pid = int(pidfile.read_text())
stat = Path(f'/proc/{pid}/stat')
assert not stat.exists() or stat.read_text().split()[2] == 'Z', 'surviving descendant'
(OUT / 'preflight.json').write_text(json.dumps({'allocation_rejected': True, 'timeout_exit': 124,
    'descendant_running_after_kill': False, 'zombie_allowed_pending_reap': True}, indent=2)+'\n')

shutil.copy(ROOT / 'main.go.txt', SCRATCH / 'main.go')
(SCRATCH / 'go.mod').write_text('module d03b\n\ngo 1.25.0\n\nrequire github.com/open-policy-agent/opa v1.19.1\n')
fetch = guarded('fetch', ['go', 'mod', 'tidy'], build=True)
if fetch['exit'] != 0:
    raise SystemExit('dependency resolution failed; see fetch channels')
for name in ('go.mod', 'go.sum'):
    shutil.copy(SCRATCH / name, OUT / name)
meta = guarded('module', ['go', 'mod', 'download', '-json', 'github.com/open-policy-agent/opa@v1.19.1'], build=True)
origin = json.loads(meta['stdout'])
assert origin['Origin']['Hash'] == '54896f9f28515d14b31a93e6ef5737bb85866e30'
guarded('graph', ['go', 'mod', 'graph'], build=True)
guarded('toolchain', ['go', 'env', '-json'], build=True)
binary = SCRATCH / 'probe'
built = guarded('build', ['go', 'build', '-p=2', '-trimpath', '-buildvcs=false', '-o', str(binary), '.'], build=True)
if built['exit'] != 0:
    raise SystemExit('build failed; see build channels')
caps = guarded('capabilities', [str(binary), 'capabilities'])
assert caps['exit'] == 0
(OUT / 'capabilities.json').write_text(caps['stdout'])

requests = []
class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        requests.append(self.path); self.send_response(200); self.end_headers()
    def log_message(self, *args):
        pass
server = http.server.HTTPServer(('127.0.0.1', 0), Handler)
threading.Thread(target=server.serve_forever, daemon=True).start()
marker = 'D03_SYNTHETIC_SECRET_ONLY'
valid = {'level': 'low', 'reasons': [{'code': 'synthetic', 'message': 'synthetic', 'paths': []}]}
V = json.dumps(valid)
PREFIX = 'package tfui.risk.v1\nimport rego.v1\n'
cases = {}
def add(name, body, inp=None, strict=True, reject=False):
    spec = {'Source': PREFIX+body, 'Input': inp or {'zero': 0, 'a': True, 'b': True, 'x': 1, 'y': 2},
            'Strict': strict, 'Reject': reject}
    assert len(spec['Source'].encode()) <= 65536
    assert len(json.dumps(spec['Input']).encode()) <= MAX
    cases[name] = spec

add('valid', f'classification := {V}')
add('undefined', 'unrelated := true')
for name, value in [('empty_array','[]'),('empty_object','{}'),('null','null'),('scalar','42'),
                    ('missing_reasons','{"level":"low"}'),('unknown_level',V.replace('"low"','"urgent"'))]:
    add(name, 'classification := '+value)
add('conflict', 'classification := input.x if input.a\nclassification := input.y if input.b')
add('parse_error', 'classification := [')
add('unknown_builtin', 'classification := nonexistent_builtin(1)')
for strict in (True, False):
    suffix = 'strict' if strict else 'nonstrict'
    add('builtin_executed_'+suffix, 'classification := 1 / input.zero', strict=strict)
    add('builtin_unreachable_'+suffix, f'classification := {V}\nunused := 1 / input.zero', strict=strict)
for name, call in {
    'http': f'http.send({{"method":"GET","url":"http://127.0.0.1:{server.server_port}/probe"}})',
    'dns': 'net.lookup_ip_addr("localhost")', 'time': 'time.now_ns()',
    'rand': 'rand.intn("synthetic", 10)', 'uuid': 'uuid.rfc4122("synthetic")',
    'runtime': 'opa.runtime()', 'trace': f'trace("{marker}")',
}.items():
    add(f'denied_{name}_executed', 'classification := '+call)
    add(f'denied_{name}_unreachable', f'classification := {V}\nunused := '+call)
for suffix, body in [('executed',f'classification := {V} if {{ print(input.token) }}'),
                     ('unreachable',f'classification := {V}\nunused if {{ print(input.token) }}')]:
    add('print_raw' if suffix=='executed' else 'print_raw_unreachable', body, {'token':marker})
    add('print_rejected' if suffix=='executed' else 'print_rejected_unreachable', body, {'token':marker},reject=True)
# Also measure explicit AST trace rejection in the candidate path.
for suffix in ('executed','unreachable'):
    spec=cases[f'denied_trace_{suffix}']
    add(f'trace_rejected_{suffix}',spec['Source'][len(PREFIX):],reject=True)
marked = {'token': {'kind': 'sensitive'}}
add('secret_marked_echo', 'classification := input', marked)
secret_result = V.replace('"message": "synthetic"', '"message": sprintf("%v", [input.token])')
add('secret_marked_interpolation', 'classification := '+secret_result, marked)
add('secret_unmarked_interpolation', 'classification := '+secret_result, {'token':marker})
add('secret_literal', 'classification := '+V.replace('"message": "synthetic"',json.dumps('message')+': '+json.dumps(marker)))
add('secret_error', f'classification := {marker}(1)')

# Freeze all modules, serialized inputs, harness and capabilities before any probe evaluation.
fixture_dir = OUT / 'fixtures'; fixture_dir.mkdir(exist_ok=True)
for name,spec in cases.items():
    (fixture_dir / (name+'.json')).write_text(json.dumps(spec, indent=2)+'\n')
    (fixture_dir / (name+'.rego')).write_text(spec['Source']+'\n')
files = [*ROOT.glob('*.py'), ROOT/'main.go.txt', ROOT/'protocol.md', OUT/'capabilities.json',
         OUT/'go.mod', OUT/'go.sum', *fixture_dir.iterdir()]
hashes = {str(p.relative_to(ROOT)): hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(files)}
source_root = Path(origin['Dir'])
for sub in ('v1/rego/rego.go','v1/ast/capabilities.go','v1/ast/compile.go'):
    hashes['opa/'+sub] = hashlib.sha256((source_root/sub).read_bytes()).hexdigest()
(OUT/'frozen-sha256.json').write_text(json.dumps(hashes,indent=2)+'\n')
results = {}
try:
    for name in cases:
        result_path = SCRATCH / (name+'.result.json')
        rec = guarded(name, [str(binary), str(fixture_dir/(name+'.json')), str(result_path)])
        observation = json.loads(result_path.read_text()) if result_path.exists() else {}
        observation.update(rec)
        # Raw errors stay in evidence files; stdout/stderr remain literal captured streams.
        results[name] = observation
finally:
    server.shutdown(); server.server_close()
(OUT/'results.json').write_text(json.dumps({'cases':results,'http_requests':len(requests),
    'environment':ENV, 'scratch':str(SCRATCH), 'host':os.uname().sysname+' '+os.uname().release,
    'scope':'standalone boundary controls, no production evaluator or cost claim'},indent=2)+'\n')
print(f'{len(results)} cases recorded in {OUT}; loopback requests={len(requests)}')
