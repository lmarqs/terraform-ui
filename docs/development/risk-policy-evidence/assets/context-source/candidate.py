"""Disposable D-02c source identity model; no production evaluator or HCL integration."""
from dataclasses import dataclass
import fnmatch
import hashlib
import os
from pathlib import Path
from types import MappingProxyType


@dataclass(frozen=True)
class Selection:
    path: object
    origin: object


def select(root_dir, root_selector=None, member_dir=None, member_selector=None,
           workspaces=(), workspace=''):
    root_dir = Path(os.path.abspath(root_dir))
    member_dir = Path(os.path.abspath(member_dir or root_dir))
    selected, origin = root_selector, root_dir
    if member_selector is not None:
        selected, origin = member_selector, member_dir
    matches = [(pattern, value) for pattern, value in workspaces
               if pattern == workspace or fnmatch.fnmatchcase(workspace, pattern)]
    exact = [entry for entry in matches if entry[0] == workspace]
    if workspace and matches:
        value = (exact or matches)[0][1]
        if value is not None:
            selected, origin = value, member_dir
    if selected is None or selected == '':
        return Selection(None, None)
    if not isinstance(selected, str) or not selected.strip():
        raise ValueError('invalid selector')
    return Selection(Path(os.path.abspath(origin / selected)), origin)


@dataclass(frozen=True)
class Snapshot:
    modules: object
    digest: str


def _read(path):
    # Lexical absolute paths keep symlinks visible to the component check.
    path = Path(os.path.abspath(path))
    for part in (path, *path.parents):
        if part.is_symlink():
            raise ValueError('symlink source')
    if path.is_dir():
        entries = sorted(path.iterdir(), key=lambda p: p.name.encode('utf-8'))
        files = [entry for entry in entries
                 if entry.name.endswith('.rego') and not entry.name.endswith('_test.rego')
                 and (entry.is_symlink() or not entry.is_dir())]
    else:
        files = [path]
    modules = {}
    for entry in files:
        entry.name.encode('utf-8', errors='strict')
        if (entry.is_symlink() or not entry.is_file() or entry.suffix != '.rego'
                or entry.name.endswith('_test.rego')):
            raise ValueError('invalid module')
        modules[entry.name] = entry.read_bytes()
    if not modules:
        raise ValueError('no modules')
    return modules


def snapshot(path):
    try:
        modules = _read(path)
        if modules != _read(path):
            raise ValueError('source changed during read')
    except (OSError, UnicodeError) as exc:
        raise ValueError('unreadable source') from exc
    frame = b'tfui-risk-source-v1\0' + len(modules).to_bytes(8, 'big')
    for name, source in sorted(modules.items(), key=lambda item: item[0].encode('utf-8')):
        encoded = name.encode('utf-8')
        frame += len(encoded).to_bytes(8, 'big') + encoded
        frame += len(source).to_bytes(8, 'big') + source
    return Snapshot(MappingProxyType(modules), 'sha256:' + hashlib.sha256(frame).hexdigest())


@dataclass(frozen=True)
class Ticket:
    generation: int
    context: object
    plan_id: str
    path: Path
    snapshot: Snapshot


class Gate:
    def __init__(self, resolve=None):
        self.resolve = resolve
        self.generation = 0
        self.result = None
        self.current = None

    def begin(self, context, plan_id, source_path):
        self.generation += 1
        self.result = None
        self.current = None
        ticket = Ticket(self.generation, tuple(context), plan_id,
                        Path(os.path.abspath(source_path)), snapshot(source_path))
        self.current = ticket
        return ticket

    def publish(self, ticket, result):
        if ticket is not self.current or ticket.generation != self.generation:
            return False
        try:
            if self.resolve is not None and Path(os.path.abspath(self.resolve())) != ticket.path:
                self.result = None
                return False
            if snapshot(ticket.path).digest != ticket.snapshot.digest:
                self.result = None
                return False
        except ValueError:
            self.result = None
            return False
        self.result = result
        return True
