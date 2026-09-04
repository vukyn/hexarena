---
name: vn-diacritic-search
description: "VN accent-insensitive search = kuery text.FoldVN + denormalized *_search column synced in BeforeAppendModel; đ→d gotcha; query folded server-side, FE unchanged"
metadata: 
  node_type: memory
  type: reference
---

Vietnamese diacritic-insensitive search (type no-dấu → match accented data) pattern, shipped in rainy #85.

**Helper:** `github.com/vukyn/kuery/text.FoldVN(s string) string` (v1.25.0) — NFD-decompose + strip combining marks (unicode.Mn) + **đ/Đ→d** + lowercase + trim. e.g. "Bản Tình Ca"→"ban tinh ca", "Đạt G"→"dat g". **Gotcha: đ/Đ (U+0111/0110) are standalone letters, NFD does NOT decompose them — must map explicitly** before NFD, else "đông" stays "đông".

**Approach (chosen over query-time unaccent fn):** denormalized search column, NOT a custom SQLite function (registering scalar fns on sqliteshim/modernc per-connection is fragile). Per searchable field add a real writable column `title_search`/`name_search`, set it in the entity's `BeforeAppendModel` hook (same place as timestamps) = `FoldVN(Title/Name)` on every write. GetList filter: `<col>_search LIKE '%'+FoldVN(query)+'%'` — both sides folded+lowercased so plain LIKE works (no LOWER()). FE sends raw query unchanged; backend folds it.

**rainy domains done:** track.title_search, album.title_search, artist.name_search, station.name_search. Migration 019 = ADD COLUMN ×4 + backfill existing rows via FoldVN (migration imports kuery/text, loops rows). Down = DROP COLUMN.

**Notes:** substring `%q%` search already full-scans (no index benefit either way) — column chosen for portability/no-driver-couple, not speed. Partial `.Set("col=?")` updates don't touch *_search (only named cols written) so the empty-struct FoldVN("") in the hook is harmless. Exact-match find-or-create paths (FindByName/FindByTitle) left on LOWER()=LOWER() — out of scope (not substring search). Reusable in isme. See [[postgres-sqlite-gotchas]], [[kuery-shared-lib-rule]].
