# Checks the agents layout of the repository it runs in and reports each gap with the step that closes it.
# Run from the repository root: sh .agents/skills/agents-layout/scripts/check.sh
# Writes nothing. Prints one line per check: "ok <what>", "WARN <what> -- <next>" or "FAIL <what> -- <next>".
# A FAIL is a setup a harness cannot read; a WARN is a recommendation. Exits 1 when any check fails.

status=0
ok() { printf 'ok %s\n' "$1"; }
warn() { printf 'WARN %s -- %s\n' "$1" "$2"; }
fail() { printf 'FAIL %s -- %s\n' "$1" "$2"; status=1; }

root=$(git rev-parse --show-toplevel 2>/dev/null) || {
  fail 'not inside a git repository' 'run this from the repository root'
  exit 1
}
cd "$root" || exit 1

newline='
'

check_agents_md() {
  if grep -q 'This file is an unfilled template' AGENTS.md; then
    warn "AGENTS.md still has the template's note" \
      'fill the file in as the note says, then delete the note once it is all this script reports'
  else
    ok "AGENTS.md has no template note"
  fi

  fills=$(grep -c '\[\[fill:' AGENTS.md)
  if [ "$fills" -gt 0 ]; then
    warn "AGENTS.md has $fills lines with [[fill: blanks" \
      "grep -n '\\[\\[fill:' AGENTS.md lists them; fill each in, or write none where it does not apply"
  else
    ok 'AGENTS.md has no [[fill: blanks'
  fi

  words=$(($(wc -w < AGENTS.md)))
  if [ "$words" -gt 1200 ]; then
    warn "AGENTS.md has $words words, over 1,200" \
      'move a section into a skill in .agents/skills/ or a file in .agents/memory/ named under Deep dive'
  elif [ "$words" -gt 800 ]; then
    warn "AGENTS.md has $words words, over 800" \
      'cut lines whose removal would cause no mistake, or move a section into a skill or a memory file'
  else
    ok "AGENTS.md has $words words"
  fi

  # Inline code outside fenced blocks: a token that holds a slash and only path characters may name a path. It may
  # also be a git ref or a file made at run time, so a miss warns and the reader decides.
  paths=$(awk -F'`' '
    /^[[:space:]]*(```|~~~)/ { fenced = !fenced; next }
    !fenced { for (i = 2; i <= NF; i += 2) if ($i ~ /\// && $i ~ /^[A-Za-z0-9._\/-]+$/ && !seen[$i]++) print $i }
  ' AGENTS.md)
  missing=0
  saved_ifs=$IFS
  IFS=$newline
  for path in $paths; do
    relative=${path#/}
    if [ ! -e "$relative" ] && [ -z "$(git ls-files -- "$relative")" ]; then
      warn "AGENTS.md names $path, which is not in the tree" \
        'if it is a path, correct it or remove the line; a git ref or a file made at run time is fine'
      missing=1
    fi
  done
  IFS=$saved_ifs
  [ "$missing" -eq 0 ] && ok 'every path AGENTS.md names exists'
}

check_agents_bytes() {
  bytes=0
  files=$(git ls-files | grep -E '(^|/)AGENTS\.md$')
  saved_ifs=$IFS
  IFS=$newline
  for file in $files; do
    [ -f "$file" ] && bytes=$((bytes + $(wc -c < "$file")))
  done
  IFS=$saved_ifs
  if [ "$bytes" -gt 16384 ]; then
    warn "the tracked AGENTS.md files hold $bytes bytes, over 16 KiB" \
      'move sections into skills or memory files; Codex stops reading the chain at 32 KiB'
  else
    ok "the tracked AGENTS.md files hold $bytes bytes"
  fi
}

check_claude_files() {
  # Claude reads a CLAUDE.md in the directory of an AGENTS.md or one above, ignored or not; one anywhere else, such as
  # in an installed dependency, hides nothing.
  directories=$(git ls-files | grep -E '(^|/)AGENTS\.md$' | while IFS= read -r file; do
    directory=$(dirname "$file")
    while :; do
      printf '%s\n' "$directory"
      [ "$directory" = . ] && break
      directory=$(dirname "$directory")
    done
  done | sort -u)
  found=0
  saved_ifs=$IFS
  IFS=$newline
  for directory in $directories; do
    for name in CLAUDE.md CLAUDE.local.md .claude/CLAUDE.md; do
      file=${directory#.}/$name
      file=${file#/}
      [ -e "$file" ] || continue
      fail "$file exists, and Claude skips AGENTS.md while it does" \
        'move what the team needs into AGENTS.md, then delete it'
      found=1
    done
  done
  IFS=$saved_ifs
  [ "$found" -eq 0 ] && ok 'no CLAUDE.md, .claude/CLAUDE.md or CLAUDE.local.md beside or above an AGENTS.md'
}

check_links() {
  links=0
  for link in .claude/* .claude/.[!.]* .codex/* .codex/.[!.]*; do
    [ -L "$link" ] || continue
    links=$((links + 1))
    mode=$(git ls-files -s -- "$link" | awk '{ print $1 }')
    # A tracked link stays in every commit whatever .gitignore says, so only an untracked one can be hidden by it.
    if [ "$mode" != 120000 ]; then
      fail "$link is not tracked as a link" \
        "git add $link; when git add skips it, git check-ignore -v --no-index $link names the rule that hides it"
    elif [ ! -e "$link" ]; then
      fail "$link does not resolve" 'cook #connect again with every harness the team uses'
    else
      ok "$link is a tracked link that resolves"
    fi
  done
  if [ "$links" -eq 0 ]; then
    fail 'no links in .claude/ or .codex/' 'cook #connect with every harness the team uses, in the commit that seeds'
  fi
}

check_shared_entries() {
  # Harnesses reach .agents/ only through the shared directories and the adapters; an entry beside them is found
  # only through a path someone wrote down, and a reader of the layout does not look for it.
  entries=$(git -c core.quotePath=false ls-files -- .agents | sed -n 's|^\.agents/\([^/]*\).*|\1|p' | sort -u)
  next='move it beside what uses it in skills/, commands/, hooks/ or memory/, merge permission rules into'
  next="$next permission/policy.yaml, or move it out of .agents/ when no agent reads it; then update the paths"
  next="$next that name it"
  found=0
  saved_ifs=$IFS
  IFS=$newline
  for entry in $entries; do
    case $entry in
      README.md | permission | adapters | skills | commands | hooks | memory) ;;
      *)
        warn ".agents/$entry is outside the shared directories" "$next"
        found=1
        ;;
    esac
  done
  IFS=$saved_ifs
  [ "$found" -eq 0 ] && ok 'every tracked entry in .agents/ belongs to the layout'
}

if [ -f AGENTS.md ]; then
  check_agents_md
else
  fail 'AGENTS.md is missing' 'cook #seed, or restore AGENTS.md'
fi
check_agents_bytes
check_claude_files
check_links
check_shared_entries

exit "$status"
