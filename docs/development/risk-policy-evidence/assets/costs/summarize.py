"""Render medians and complete ranges without dropping failed trials."""
import json
from pathlib import Path
import statistics

ROOT = Path(__file__).resolve().parent
results = json.loads((ROOT/'output/results.json').read_text())


def spread(values, scale=1):
    vals = [x/scale for x in values]
    return f'{statistics.median(vals):.3f} [{min(vals):.3f}, {max(vals):.3f}]'


lines = ['# D-03c measured costs', '', 'Median [minimum, maximum]; all five samples retained. Bytes are uncompressed stripped executables.', '',
         '| Target | Variant | Phase | Successful / trials | Binary bytes | Wall seconds | Peak RSS MiB |',
         '| --- | --- | --- | --- | --- | --- | --- |']
for row in results['builds']:
    good = [s for s in row['samples'] if s['status'] == 'ok']
    if len(good) != len(row['samples']):
        lines.append(f"| {row['target']} | {row['variant']} | {row['phase']} | {len(good)}/{len(row['samples'])} | incomplete | incomplete | incomplete |")
        continue
    lines.append('| '+' | '.join([row['target'],row['variant'],row['phase'],'5/5',
        spread([s['size_bytes'] for s in good]), spread([s['wall_seconds'] for s in good]),
        spread([s['peak_rss_kib'] for s in good],1024)])+' |')
lines += ['', '| Target | Candidate extra bytes | Increase % | Cold wall delta seconds (paired) | Warm wall delta seconds (paired) |',
          '| --- | --- | --- | --- | --- |']
for target in results['provenance']['targets']:
    rows = {(r['variant'],r['phase']):r['samples'] for r in results['builds'] if r['target']==target}
    if len(rows)!=4 or any(s['status']!='ok' for row in rows.values() for s in row):
        lines.append(f'| {target} | incomplete | incomplete | incomplete | incomplete |'); continue
    base=rows['baseline','cold'][0]['size_bytes'];cand=rows['candidate','cold'][0]['size_bytes']
    delta=[]
    for phase in ('cold','warm'):
        delta.append(spread([c['wall_seconds']-b['wall_seconds'] for b,c in zip(rows['baseline',phase],rows['candidate',phase])]))
    lines.append(f'| {target} | {cand-base} | {(cand/base-1)*100:.2f} | {delta[0]} | {delta[1]} |')
lines += ['', 'Runtime is native Linux/amd64. Times below are milliseconds except validated per-Eval microseconds. Whole-process RSS includes runtime/library/fixture overhead.', '',
          '| Variant | N | Phase | Process ms | Input decode/generation ms | Source parse ms | Prepare ms | First measured Eval µs | Total Eval ms | Per Eval µs | Peak RSS MiB |',
          '| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |']
for row in results['runtime']:
    samples=row['samples']; good=[s for s in samples if s['status']=='ok']
    if len(good)!=5:
        lines.append(f"| {row['variant']} | {row['workload_n']} | {row['phase']} | incomplete | | | | | | | |"); continue
    metrics=[s['metrics'] for s in good]
    columns=[row['variant'], str(row['workload_n']), row['phase'],spread([s['wall_seconds'] for s in good],.001)]
    for field,scale in [('parse_ns',1e6),('source_parse_ns',1e6),('prepare_ns',1e6),('first_eval_ns',1e3),('eval_total_ns',1e6)]:
        columns.append(spread([m.get(field,0) for m in metrics],scale))
    columns += [spread([m['eval_total_ns']/m['correct_count'] for m in metrics],1e3),spread([s['peak_rss_kib'] for s in good],1024)]
    lines.append('| '+' | '.join(columns)+' |')
(ROOT/'output/summary.md').write_text('\n'.join(lines)+'\n')
print(ROOT/'output/summary.md')
