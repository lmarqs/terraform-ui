"""D-03c disposable build/runtime experiment. Run from any CWD on Linux/amd64."""
import hashlib
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import guard

ROOT = Path(__file__).resolve().parent
REPO = ROOT.parents[4]
OUT = ROOT / 'output'
OUT.mkdir(exist_ok=True)
SCRATCH = Path(tempfile.mkdtemp(prefix='tfui-d03c-', dir='/tmp'))
GOROOT = '/home/lmarqs/.local/share/mise/installs/go/1.25.5'
BASE = '60167777494b622ca55a203f05deb8e127357d23'
TARGETS = ['linux/amd64', 'linux/arm64', 'darwin/amd64', 'darwin/arm64']
ENV = {'PATH': f'{GOROOT}/bin:/usr/bin:/bin', 'GOROOT': GOROOT,
       'GOTOOLCHAIN': 'local', 'CGO_ENABLED': '0', 'GOMAXPROCS': '2',
       'GOPATH': str(SCRATCH/'gopath'), 'GOMODCACHE': str(SCRATCH/'modcache'),
       'GOCACHE': str(SCRATCH/'setupcache'), 'LANG': 'C.UTF-8', 'TZ': 'UTC'}
guard.OUT = OUT
guard.ENV = ENV
guard.SCRATCH = SCRATCH
results = {'provenance': {'baseline_revision': BASE, 'targets': TARGETS,
    'graphs': {}, 'hashes': {}, 'environment': ENV, 'scratch': str(SCRATCH)},
    'builds': [], 'runtime': [], 'controls': {}, 'setup': [], 'cancellation': []}


def save():
    (OUT/'results.json').write_text(json.dumps(results, indent=2)+'\n')


def run(name, args, cwd=None, env=None, build=True, **kwargs):
    guard.SCRATCH = cwd or SCRATCH
    guard.ENV = ENV | (env or {})
    rec = guard.guarded(name, args, build=build, **kwargs)
    print(name, rec['exit'], round(rec['wall_seconds'], 3), flush=True)
    return rec


def setup(name, args, cwd=None):
    rec = run(name, args, cwd)
    results['setup'].append(rec)
    save()
    if rec['status'] != 'ok':
        raise SystemExit(f'{name} failed; retained raw evidence')
    return rec


# Preflight before OPA or build execution; no inherited credentials or HOME.
mem = run('preflight-memory', ['/usr/bin/python3', '-c',
    'try:\n x=bytearray(512*1024*1024)\nexcept MemoryError:\n print("MemoryError"); raise SystemExit(42)'],
    build=False, memory=134217728)
assert mem['exit'] == 42 and 'MemoryError' in mem['stdout']
pidfile = SCRATCH/'descendant.pid'
script = ('import subprocess,time,pathlib; p=subprocess.Popen(["/usr/bin/python3","-c","import time; time.sleep(20)"]); '
          f'pathlib.Path({str(pidfile)!r}).write_text(str(p.pid)); time.sleep(20)')
timed = run('preflight-timeout', ['/usr/bin/python3', '-c', script], build=False, wall=1)
assert timed['exit'] == 124 and not timed['surviving_running_group_members']
stat = Path('/proc')/pidfile.read_text()/'stat'
assert not stat.exists() or stat.read_text().split()[2] == 'Z'
results['controls'] = {'allocation_rejected': True, 'descendant_running_after_kill': [],
                       'memory': mem, 'timeout': timed}

# Immutable archive, without checking out or editing the live module.
archive = subprocess.run(['git', 'archive', BASE], cwd=REPO, check=True, capture_output=True).stdout
for variant in ('baseline', 'candidate'):
    tree = SCRATCH/variant
    tree.mkdir()
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        tar.extractall(tree, filter='data')
probe = (ROOT/'probe.go.txt').read_text()
candidate = SCRATCH/'candidate'
mainfile = candidate/'cmd/tfui/main.go'
original = mainfile.read_text()
patched = original.replace('func main() {', 'func main() {\n if len(os.Args)>1 && os.Args[1]=="__d03" { d03Main(os.Args[2:]); return }', 1)
mainfile.write_text(patched)
(candidate/'cmd/tfui/d03_probe.go').write_text(probe.replace('func main() { d03Main(os.Args[1:]) }', ''))
# Include the exact live capability bytes used in D-03b, not the unrestricted registry.
caps = ROOT.parent/'boundary/output/capabilities.json'
shutil.copy(caps, OUT/'capabilities.json')
setup('candidate-add', ['go', 'mod', 'edit', '-require=github.com/open-policy-agent/opa@v1.19.1'], candidate)
for variant in ('baseline', 'candidate'):
    tree = SCRATCH/variant
    setup(variant+'-fetch', ['go', 'mod', 'tidy'], tree)
    setup(variant+'-prefetch', ['go', 'mod', 'download', 'all'], tree)
    graph = setup(variant+'-graph', ['go', 'mod', 'graph'], tree)
    results['provenance']['graphs'][variant] = graph['stdout']
    for file in ('go.mod', 'go.sum'):
        shutil.copy(tree/file, OUT/(variant+'-'+file))
origin = json.loads(setup('opa-origin', ['go', 'mod', 'download', '-json', 'github.com/open-policy-agent/opa@v1.19.1'], candidate)['stdout'])
assert origin['Origin']['Hash'] == '54896f9f28515d14b31a93e6ef5737bb85866e30'
results['provenance']['opa_origin'] = origin
results['provenance']['toolchain'] = json.loads(setup('toolchain', ['go', 'env', '-json'])['stdout'])
results['provenance']['go_version'] = setup('go-version', ['go', 'version'])['stdout']
results['provenance']['host'] = {'uname': list(os.uname()),
    'cpu': Path('/proc/cpuinfo').read_text(), 'memory': Path('/proc/meminfo').read_text()}
patch = subprocess.run(['diff', '-u', str(SCRATCH/'baseline/cmd/tfui/main.go'), str(mainfile)], capture_output=True, text=True)
(OUT/'candidate-main.patch').write_text(patch.stdout)
for file in [*ROOT.glob('*.py'), ROOT/'probe.go.txt', ROOT/'fixtures.json', ROOT/'protocol.md',
             OUT/'capabilities.json', OUT/'candidate-main.patch', *OUT.glob('*-go.mod'), *OUT.glob('*-go.sum')]:
    results['provenance']['hashes'][str(file.relative_to(ROOT))] = hashlib.sha256(file.read_bytes()).hexdigest()
results['provenance']['hashes']['go-compiler'] = hashlib.sha256(Path(GOROOT+'/bin/go').read_bytes()).hexdigest()
(OUT/'frozen-sha256.json').write_text(json.dumps(results['provenance']['hashes'],indent=2)+'\n')
save()

flags = ['go', 'build', '-p=2', '-trimpath', '-buildvcs=false', '-ldflags=-s -w -X main.version=d03-experiment']
native = {}
for target in TARGETS:
    goos, goarch = target.split('/')
    key = target.replace('/', '-')
    for phase in ('cold', 'warm'):
        rows = {v: {'target':target, 'variant':v, 'phase':phase, 'samples':[]} for v in ('baseline','candidate')}
        if phase == 'warm':
            for variant in rows:
                setup_env = {'GOOS':goos, 'GOARCH':goarch, 'GOCACHE':str(SCRATCH/f'cache-{key}-{variant}-warm')}
                prime = run(f'{key}-{variant}-prime', [*flags, '-o', str(SCRATCH/'prime'), './cmd/tfui'], SCRATCH/variant, setup_env)
                rows[variant]['prime'] = prime
        for i in range(5):
            for variant in (('baseline','candidate') if i%2 == 0 else ('candidate','baseline')):
                cache = SCRATCH/f'cache-{key}-{variant}-{phase}'
                if phase == 'cold':
                    cache = SCRATCH/f'cache-{key}-{variant}-cold-{i}'
                artifact = SCRATCH/f'{key}-{variant}'
                if artifact.exists(): artifact.unlink()
                rec = run(f'{key}-{variant}-{phase}-{i}', [*flags, '-o', str(artifact), './cmd/tfui'],
                          SCRATCH/variant, {'GOOS':goos, 'GOARCH':goarch, 'GOCACHE':str(cache)})
                if rec['status'] == 'ok':
                    rec['size_bytes'] = artifact.stat().st_size
                    rec['sha256'] = hashlib.sha256(artifact.read_bytes()).hexdigest()
                    if target == 'linux/amd64':
                        shutil.copy(artifact, SCRATCH/f'native-{variant}')
                        native[variant] = SCRATCH/f'native-{variant}'
                rows[variant]['samples'].append(rec)
                save()
                if phase == 'cold': shutil.rmtree(cache, ignore_errors=True)
        results['builds'].extend(rows.values())
        save()
        if phase == 'warm':
            for variant in rows: shutil.rmtree(SCRATCH/f'cache-{key}-{variant}-warm',ignore_errors=True)

# Standalone same source and candidate graph; compilation is not a release-target comparison.
standalone = candidate/'d03standalone'
standalone.mkdir()
(standalone/'main.go').write_text(probe)
setup('standalone-build', [*flags, '-o', str(SCRATCH/'standalone'), './d03standalone'], candidate)
for variant in ('candidate', 'standalone', 'overhead'):
    binary = native['candidate'] if variant == 'candidate' else SCRATCH/'standalone'
    prefix = [str(binary), '__d03'] if variant == 'candidate' else [str(binary)]
    for n in (1, 100, 1000):
        for phase in ('cold', 'warm'):
            row = {'target':'linux/amd64','variant':variant,'workload_n':n,'phase':phase,'samples':[]}
            for i in range(5):
                rec = run(f'runtime-{variant}-{n}-{phase}-{i}', [*prefix, str(ROOT/'fixtures.json'), str(OUT/'capabilities.json'), str(n), phase, variant], build=False)
                if rec['status'] == 'ok': rec['metrics'] = json.loads(rec['stdout'])
                row['samples'].append(rec)
            results['runtime'].append(row)
            save()
for mode in ('cancelled','deadline'):
    rec = run('bounded-'+mode, [str(native['candidate']), '__d03', str(ROOT/'fixtures.json'), str(OUT/'capabilities.json'), '1', 'cold', 'candidate', mode], build=False)
    results['cancellation'].append({'mode':mode, **rec})
save()
print('Recorded complete experiment:', OUT, flush=True)
