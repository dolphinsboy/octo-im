# wkdb Layout

`pkg/wkdb` now uses explicit versioned subdirectories so the legacy and new
implementations are separated by path.

- `v2/`: legacy `wkdb` implementation. The package name is still `wkdb`, but
  the import path is `github.com/WuKongIM/WuKongIM/pkg/wkdb/v2`.
- `v3/`: slot-native `wkdb` implementation and the long-term replacement
  target.
- `v3/docs/`: `wkdb v3` architecture notes, migration status, and replacement
  plan.

Important context:

- the runtime is still hybrid today; see `pkg/wkdb/v3/docs/MIGRATION_STATUS.md`
- removing legacy `v2` still requires the work listed in
  `pkg/wkdb/v3/docs/LEGACY_REPLACEMENT_PLAN.md`
