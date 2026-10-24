#!/usr/bin/env python3
"""Check the fictional T00 design examples, without collecting host information."""
import hashlib
import json
from pathlib import Path
import re
from datetime import datetime

BASE = Path(__file__).resolve().parent
ROOT = BASE.parents[3]
REVISION = 't00-2026-10-24.1'


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        assert key not in result, f'duplicate JSON key: {key}'
        result[key] = value
    return result


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False)


def digest(value):
    return 'sha256:' + hashlib.sha256(canonical(value).encode('utf-8')).hexdigest()


def timestamp(value):
    assert re.fullmatch(r'\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z', value)
    datetime.strptime(value, '%Y-%m-%dT%H:%M:%SZ')


def selector(finding):
    return {key: finding[key] for key in ('collector', 'kind', 'identity')}


def by_port(snapshot, port):
    return next(f for f in snapshot['findings'] if f['identity'].get('local_port') == port)


artifacts = {p.name: json.loads(p.read_text(), object_pairs_hook=unique_object)
             for p in BASE.glob('*.json')}
for name, obj in artifacts.items():
    if obj.get('kind') not in ('snapshot', 'assessments', 'review'):
        continue
    assert obj['schema_version'] == '0.1' and obj['contract_revision'] == REVISION, name
    assert obj['target_id'] == 'fictional-web-01', name
    if obj['kind'] == 'snapshot':
        timestamp(obj['collected_at'])
        timestamp(obj['finished_at'])
        assert obj['finished_at'] >= obj['collected_at']
        assert obj['tool_version'] == 'fictional-not-implemented'
        evidence = {e['evidence_id']: e for e in obj['evidence']}
        assert len(evidence) == len(obj['evidence']), name
        assert len({f['finding_id'] for f in obj['findings']}) == len(obj['findings']), name
        assert obj['findings'] == sorted(obj['findings'], key=lambda f: f['finding_id'])
        for e in obj['evidence']:
            timestamp(e['collected_at'])
            assert e['method'] == 'fictional fixture, no host read'
        for f in obj['findings']:
            assert f['finding_id'] == digest(['avera.identity.v1', obj['target_id'],
                                              f['collector'], f['kind'], f['identity']]), name
            assert f['fact_fingerprint'] == digest(['avera.facts.v1', f['kind'],
                                                    f['identity'], f['facts']]), name
            assert f['evidence_ids'] and set(f['evidence_ids']) <= evidence.keys(), name
            assert not {'decision', 'rationale', 'actor', 'assessed_at'} & f.keys(), name
            if f['kind'] == 'network.endpoint':
                assert evidence[f['evidence_ids'][0]]['values']['observed'] == f['facts']
    elif obj['kind'] == 'assessments':
        ids = [a['assessment_id'] for a in obj['entries']]
        assert ids == sorted(set(ids)), name
        for a in obj['entries']:
            if a['decision'] != 'unreviewed':
                assert a['actor'].strip() and a['rationale'].strip(), name
                timestamp(a['assessed_at'])
            if a['origin'] == 'reviewed_snapshot':
                snapshots = [s for s in artifacts.values()
                             if s.get('snapshot_id') == a['basis']['snapshot_id']
                             and s.get('kind') == 'snapshot']
                if snapshots:
                    matches = [f for f in snapshots[0]['findings'] if selector(f) == a['selector']]
                    # The deliberately unmatched annotation must still have an authentic historical basis.
                    assert len(matches) == 1, (name, a['assessment_id'])
                    assert matches[0]['fact_fingerprint'] == a['basis']['fact_fingerprint']

for vector in artifacts['normalization-vectors.json']['vectors']:
    for field, expected in [('identity_canonical', 'finding_id'), ('facts_canonical', 'fact_fingerprint')]:
        preimage = vector[field]
        assert canonical(json.loads(preimage)) == preimage
        assert 'sha256:' + hashlib.sha256(preimage.encode()).hexdigest() == vector[expected]

encoding = artifacts['normalization-vectors.json']['encoding_vector']
assert canonical(encoding['value']) == encoding['canonical']
assert digest(encoding['value']) == encoding['digest']
assert {v['finding_id'] for v in artifacts['normalization-vectors.json']['vectors']} == {
    f['finding_id'] for f in artifacts['snapshot-first.json']['findings']}

cases = {}
for scenario in artifacts['scenarios.json']['scenarios']:
    current = artifacts[scenario['snapshot']]
    previous = artifacts[scenario['previous']] if scenario['previous'] else None
    aa = artifacts[scenario['assessments']]['entries'] if scenario['assessments'] else []
    r = artifacts[scenario['expected']]
    cases[scenario['name']] = r
    assert r['snapshot_id'] == current['snapshot_id']
    assert r['previous_snapshot_id'] == (previous['snapshot_id'] if previous else None)
    assert r['assessment_ids'] == sorted(a['assessment_id'] for a in aa)
    assert r['coverage_statuses'] == {c['collector']: c['status'] for c in current['coverage']}
    now = {f['finding_id']: f for f in current['findings']}
    before = {f['finding_id']: f for f in previous['findings']} if previous else {}
    assert {i['finding_id'] for i in r['items']} == now.keys() | before.keys()
    assert len(r['items']) == len(now.keys() | before.keys())
    linked = set()
    for item in r['items']:
        fid = item['finding_id']
        is_current = fid in now
        assert item['presence'] == ('current' if is_current else 'historical')
        matches = [a for a in aa if is_current and a['selector'] == selector(now[fid])]
        assert item['assessment_ids'] == sorted(a['assessment_id'] for a in matches)
        linked.update(item['assessment_ids'])
        if not is_current:
            assert item['change'] in ('unknown', 'not_observed')
        elif not previous:
            assert item['change'] == 'not_compared'
        elif fid not in before:
            assert item['change'] == 'new'
        elif now[fid]['fact_fingerprint'] != before[fid]['fact_fingerprint']:
            assert item['change'] == 'changed'
        else:
            assert item['change'] in ('unchanged', 'unknown')
        if len(matches) > 1:
            assert item['applicability'] == 'conflict'
            assert item['reason_codes'] == ['multiple_assessments']
        if item['applicability'] == 'applicable':
            assert len(matches) == 1 and is_current
            a = matches[0]
            facts = now[fid]['facts']
            assert r['coverage_statuses'][now[fid]['collector']] == 'complete'
            if now[fid]['kind'] == 'network.endpoint':
                assert facts['attribution_complete']
            else:
                assert facts['interpretation_complete'] and facts['local_context_complete']
            if a['origin'] == 'advance':
                assert a['basis']['expectation']['facts'] == facts
            else:
                assert a['basis']['fact_fingerprint'] == now[fid]['fact_fingerprint']
            assert item['current_decision'] == a['decision']
        else:
            assert item['current_decision'] == 'unreviewed'
    assert r['unmatched_assessment_ids'] == sorted(set(r['assessment_ids']) - linked)
    current_items = [i for i in r['items'] if i['presence'] == 'current']
    counts = r['summary']
    assert counts['current'] == len(current_items) == len(now)
    assert counts['historical'] == len(r['items']) - len(current_items)
    for decision in ('accepted', 'action_required', 'unreviewed'):
        assert counts[decision] == sum(i['current_decision'] == decision for i in current_items)
    for applicability in ('needs_review', 'conflict'):
        assert counts[applicability] == sum(i['applicability'] == applicability for i in current_items)
    assert counts['unmatched_assessments'] == len(r['unmatched_assessment_ids'])
    assert counts['current'] == sum(counts[d] for d in ('accepted', 'action_required', 'unreviewed'))

first = artifacts['snapshot-first.json']
repeat = artifacts['snapshot-repeat.json']
changed = artifacts['snapshot-changed.json']
http = by_port(first, 80)
http_id = http['finding_id']
item = lambda case: next(i for i in cases[case]['items'] if i['finding_id'] == http_id)
assert first['evidence'] != repeat['evidence']
assert first['findings'] == repeat['findings'], 'PID/time-only changes must not change hashes'
assert by_port(changed, 80)['finding_id'] == http_id
assert by_port(changed, 80)['fact_fingerprint'] != http['fact_fingerprint']
assert cases['first']['summary']['unreviewed'] == 4
assert item('unchanged')['applicability'] == 'applicable'
assert item('changed')['applicability'] == 'needs_review'
assert cases['changed']['summary']['accepted'] == 0
assert item('reconfirmed')['current_decision'] == 'action_required'
assert item('reconfirmed')['change'] == 'changed'
assert item('advance')['applicability'] == 'applicable'
assert item('basis-only')['applicability'] == 'applicable'
assert item('insufficient')['applicability'] == 'needs_review'
assert item('partial')['applicability'] == 'needs_review'
assert item('conflict')['applicability'] == 'conflict'
assert cases['advance-sudo']['summary']['accepted'] == 1
assert cases['unmatched']['summary']['unmatched_assessments'] == 1
assert cases['failed']['summary']['current'] == cases['empty']['summary']['current'] == 0
assert set(cases['failed']['coverage_statuses'].values()) == {'failed'}
assert set(cases['empty']['coverage_statuses'].values()) == {'complete'}
assert {i['change'] for i in cases['failed']['items']} == {'unknown'}
assert {i['change'] for i in cases['empty']['items']} == {'not_observed'}
old = next(a for a in artifacts['assessments-reviewed.json']['entries'] if a['assessment_id'] == 'a-http')
new = next(a for a in artifacts['assessments-reconfirmed.json']['entries']
           if a['assessment_id'] == 'a-http-reconfirmed')
assert old['basis']['snapshot_id'] == 'fictional-s1' and old['decision'] == 'accepted'
assert old['basis'] != new['basis'] and old['rationale'] != new['rationale']
assert new['basis']['snapshot_id'] == 'fictional-s3'

# Resolve relative Markdown file links in all documents touched by this baseline.
docs = [BASE / 'README.md', * (ROOT / 'docs/specifications/prototype/examples').glob('*.md')]
docs += [ROOT / 'docs/specifications/prototype' / n for n in
         ('data-contracts.md', 'decisions.md', 'architecture.md', 'README.md', 'tasks/T00-contract-baseline.md')]
links = 0
for path in docs:
    for target in re.findall(r'\[[^\]]*\]\(([^)]+)\)', path.read_text()):
        if '://' in target or target.startswith('#'):
            continue
        assert (path.parent / target.split('#')[0]).resolve().exists(), (path, target)
        links += 1
report = (ROOT / 'docs/specifications/prototype/examples/fictional-report.md').read_text()
for case, review in cases.items():
    counts = review['summary']
    row = '| [{}]({}) | {} | {} | {} | {} | {} | {} | {} | {} |'.format(
        case, '../../../../test/fixtures/prototype/contracts/review-' + case + '.json',
        *[counts[k] for k in ('current', 'accepted', 'action_required', 'unreviewed',
                             'needs_review', 'conflict', 'historical', 'unmatched_assessments')])
    assert row in report, case
for finding in first['findings']:
    assert finding['finding_id'] in report
    for eid in finding['evidence_ids']:
        assert '`' + eid + '`' in report
print(f'Validated {len(artifacts)} JSON files, {len(cases)} scenarios, fixed hashes, history, report counts, and {links} local links.')
