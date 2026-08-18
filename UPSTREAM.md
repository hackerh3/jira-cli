# Upstream Sync Tracking

Which fork release contains which state of [ankitpokhrel/jira-cli](https://github.com/ankitpokhrel/jira-cli).

Tag namespaces are deliberately distinct: fork releases are **versionless** (`1.11.0`), upstream
releases carry the `v` prefix (`v1.7.0`). They never collide.

Upstream has not cut a release since **v1.7.0 (2025-08-30)**, so upstream state is expressed as
`v1.7.0+N`, where N counts upstream commits merged past that tag.

## Matrix

| Fork release | Date | Upstream base commit | Upstream state | Notes |
|---|---|---|---|---|
| 1.11.2 | 2026-08-17 | `1f3b2ef` | v1.7.0+15 | epic tree migrated to /search/jql (cloud 410 fix) |
| 1.11.1 | 2026-08-05 | `1f3b2ef` | v1.7.0+15 | UTF-8 wiki parser fix, JQL preserved with `--show-all-issues`, epic add/remove without admin, `--plain` implies table, Go 1.26, golangci-lint 2.12 |
| 1.11.0 | 2026-05-13 | `396933d` | v1.7.0+6 | |
| 1.10.0 | 2026-05-13 | `396933d` | v1.7.0+6 | |
| 1.9.0 | 2026-05-13 | `396933d` | v1.7.0+6 | |
| 1.8.1 | 2026-05-12 | `396933d` | v1.7.0+6 | |
| 1.8.0 | 2026-05-12 | `396933d` | v1.7.0+6 | first tracked fork release |

`396933d` = `fix: Autocomplete should work regardless of token (#939)`
`1f3b2ef` = `fix: Preserve jql filter when using --show-all-issues (#1014)`

## Regenerating

```sh
git remote add upstream https://github.com/ankitpokhrel/jira-cli.git   # once
git fetch upstream --tags

for t in $(git tag --list --sort=creatordate | grep -vE '^v') main; do
  mb=$(git merge-base "$t" upstream/main)
  printf "%-8s %s  v1.7.0+%s  %s\n" \
    "$t" "$(git rev-parse --short "$mb")" \
    "$(git rev-list --count v1.7.0.."$mb")" \
    "$(git log -1 --format=%ad --date=short "$t")"
done
```

## Checking drift

```sh
git fetch upstream
git rev-list --left-right --count upstream/main...main   # left = commits we are behind
git log --oneline main..upstream/main
```

Sync, verification, release, and this matrix are automated by the on-prem GitLab project
`hhaecker/jira-cli-sync` (gitlab.vi.vector.int); failures surface as Jira tickets in XPE3.
