"""Delivery integrity checks supplement the test-writer's premeasurement assertions."""
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent
OUT = ROOT/'output'
r = json.loads((OUT/'results.json').read_text())
for name, digest in r['provenance']['hashes'].items():
    if name == 'go-compiler':
        path = Path(r['provenance']['toolchain']['GOROOT'])/'bin/go'
    else:
        path = ROOT/name
    assert hashlib.sha256(path.read_bytes()).hexdigest() == digest, name
assert r['provenance']['opa_origin']['Origin']['Hash'] == '54896f9f28515d14b31a93e6ef5737bb85866e30'
for row in r['builds']:
    if row['phase'] == 'warm':
        prime = row['prime']
        assert prime['status'] == 'ok' and prime['external_limit'] == 'none'
        assert prime['pipes_drained'] and not prime['surviving_running_group_members']
    # Reproducible stripped artifacts within a variant/target, despite distinct cache locations.
    assert len({s['sha256'] for s in row['samples']}) == 1, (row['target'],row['variant'],row['phase'])
for record in [*r['setup'], *(s for row in r['builds']+r['runtime'] for s in row['samples']), *r['cancellation']]:
    assert record['status'] == 'ok', record['command']
    assert record['pipes_drained'] and not any(record['truncated'].values())
    assert not record['surviving_running_group_members']
assert len(r['cancellation']) == 2
for record in r['cancellation']:
    metrics = json.loads(record['stdout'])
    assert metrics['prepare_count'] == 1
    if record['mode'] == 'cancelled':
        assert metrics['outcome'] == 'eval_error' and metrics['context_cause'] == 'context canceled'
    else:
        assert metrics['outcome'] in ('success','eval_error')
        if metrics['outcome'] == 'eval_error':
            assert metrics['context_cause'] == 'context deadline exceeded'
print('Frozen hashes, pinned origin, all priming builds, artifact consistency, bounded channels/groups and cancellation observations verified.')
