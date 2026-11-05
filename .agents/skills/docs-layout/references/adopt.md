# Adopt the layout

Brings a repository's existing documentation under a map; the person reviews the result as one change.

1. Write the map with the person, by the skill's `map` procedure, unless one exists.
2. List the documentation: `docs/` and the files elsewhere that hold the same kinds of fact, such as
   `ARCHITECTURE.md` or `doc/adr/`. Agree the scope with the person.
3. Place each file by the map's questions, or split it when its parts fit different kinds. Ask where none fits; the
   answer may be outside `docs/`, a kind the team declares, or nowhere.
4. Move each page with `git mv`, so history follows it. Records keep their numbers.
5. Edit for facts and owners: keep a statement on its owner and link it from the rest. Move what describes a page
   into its front matter; keep any field a program reads, and ask before dropping any other kept by hand. Leave a
   page's shape alone unless it fails a reader.
6. Write a guide for each kind kept as a directory, fix the links to moved files, and report the moves, splits and
   open questions.
