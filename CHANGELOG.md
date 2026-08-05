## [1.11.1](https://github.com/hackerh3/jira-cli/compare/1.11.0...1.11.1) (2026-08-05)

## [1.11.0](https://github.com/hackerh3/jira-cli/compare/1.10.0...1.11.0) (2026-05-13)

### Features

* **cmd:** warn on duplicate subtask type during issue creation ([#12](https://github.com/hackerh3/jira-cli/issues/12)) ([79a0dc2](https://github.com/hackerh3/jira-cli/commit/79a0dc251ee512bcb150367ddfc028cdbe8ef633))

### Bug Fixes

* **comment:** move markdown conversion to command layer, add --raw flag ([#14](https://github.com/hackerh3/jira-cli/issues/14)) ([e34a03e](https://github.com/hackerh3/jira-cli/commit/e34a03ec0b17dfca882cfe3fcbc619508de9f51f)), closes [#13](https://github.com/hackerh3/jira-cli/issues/13)

## [1.10.0](https://github.com/hackerh3/jira-cli/compare/1.9.0...1.10.0) (2026-05-13)

### Features

* **view:** add --compact flag for LLM-friendly output ([#15](https://github.com/hackerh3/jira-cli/issues/15)) ([7cb397f](https://github.com/hackerh3/jira-cli/commit/7cb397f1efc100bd935e502e5baf2e37fc5e2e9a)), closes [#6](https://github.com/hackerh3/jira-cli/issues/6)

## [1.9.0](https://github.com/hackerh3/jira-cli/compare/1.8.1...1.9.0) (2026-05-13)

### Features

* **cmd:** add epic tree command ([#11](https://github.com/hackerh3/jira-cli/issues/11)) ([a589007](https://github.com/hackerh3/jira-cli/commit/a589007d085b9962af41d4849fc24ce81fe39648))

## [1.8.1](https://github.com/hackerh3/jira-cli/compare/1.8.0...1.8.1) (2026-05-12)

### Bug Fixes

* **ci:** flatten archive layout and build all platforms ([5cf0eca](https://github.com/hackerh3/jira-cli/commit/5cf0ecab63fc9b70358ae0f7c99bd4b9d6eac2a2))

# Changelog

All notable changes to the hackerh3/jira-cli fork are documented here.
This fork uses versionless tags (e.g. `1.8.0`) to distinguish from upstream `v1.x.0`.

## [1.8.0](https://github.com/hackerh3/jira-cli/releases/tag/1.8.0) (2026-05-12)

First release under the new versioning scheme. Incorporates all upstream
changes through v1.7.0 plus fork-exclusive features.

### Features

* **jira:** add move-project command for moving issues between projects ([e4b8e97](https://github.com/hackerh3/jira-cli/commit/e4b8e97))
* **jira:** add move-project API via JSP form wizard ([53dd911](https://github.com/hackerh3/jira-cli/commit/53dd911))
* **jira:** add JSP session client with cookie jar ([70713a3](https://github.com/hackerh3/jira-cli/commit/70713a3))
* **api:** add GetIssueComment and UpdateIssueComment ([71f6b5a](https://github.com/hackerh3/jira-cli/commit/71f6b5a))
* **cmd:** add comment edit subcommand ([e27710d](https://github.com/hackerh3/jira-cli/commit/e27710d))
* **jira:** add Deviniti Issue Template support for issue creation ([d1a9bd2](https://github.com/hackerh3/jira-cli/commit/d1a9bd2))

### Bug Fixes

* **jira:** detect broken workflow and suggest move-project workaround ([8b17182](https://github.com/hackerh3/jira-cli/commit/8b17182))
* **jira:** confirm step needs confirm=true + smart key extraction ([e28a8b8](https://github.com/hackerh3/jira-cli/commit/e28a8b8))
* resolve golangci-lint failures in move-project code ([0ec114d](https://github.com/hackerh3/jira-cli/commit/0ec114d))
* add plain output for project list ([e58dbc4](https://github.com/hackerh3/jira-cli/commit/e58dbc4))
* preserve raw JQL ordering ([3061f88](https://github.com/hackerh3/jira-cli/commit/3061f88))

### CI/CD

* semantic-release pipeline with goreleaser cross-compilation
* versionless tag format (`1.x.0`) to avoid upstream collision
