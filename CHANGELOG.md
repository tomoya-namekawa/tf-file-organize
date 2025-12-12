# Changelog

## [0.5.3](https://github.com/tomoya-namekawa/tf-file-organize/compare/v0.5.2...v0.5.3) (2025-12-12)


### Bug Fixes

* **deps:** update module github.com/spf13/cobra to v1.10.2 ([#44](https://github.com/tomoya-namekawa/tf-file-organize/issues/44)) ([eb6bfe0](https://github.com/tomoya-namekawa/tf-file-organize/commit/eb6bfe02b112b17e24b744d9261f74964e427f70))
* **deps:** update module github.com/zclconf/go-cty to v1.17.0 ([#49](https://github.com/tomoya-namekawa/tf-file-organize/issues/49)) ([7a7477f](https://github.com/tomoya-namekawa/tf-file-organize/commit/7a7477fc9061fe6461befa301784d15fb239b362))

## [0.5.2](https://github.com/tomoya-namekawa/tf-file-organize/compare/v0.5.1...v0.5.2) (2025-08-23)


### Bug Fixes

* **deps:** update module github.com/hashicorp/hcl/v2 to v2.24.0 ([#29](https://github.com/tomoya-namekawa/tf-file-organize/issues/29)) ([96ba3d0](https://github.com/tomoya-namekawa/tf-file-organize/commit/96ba3d0ceb38eafb069602545c8c9750223213cc))
* **deps:** update module github.com/zclconf/go-cty to v1.16.4 ([#37](https://github.com/tomoya-namekawa/tf-file-organize/issues/37)) ([c3b4f8b](https://github.com/tomoya-namekawa/tf-file-organize/commit/c3b4f8b387ea23538a103af3dee9e1549eb11c5a))
* resolve false "Created file" messages in idempotent runs ([#40](https://github.com/tomoya-namekawa/tf-file-organize/issues/40)) ([390844b](https://github.com/tomoya-namekawa/tf-file-organize/commit/390844b4887cb19ccb5ee79015514a526bd7d027))

## [0.5.1](https://github.com/tomoya-namekawa/tf-file-organize/compare/v0.5.0...v0.5.1) (2025-06-22)


### Bug Fixes

* improve idempotency by fixing file removal logic ([fba9204](https://github.com/tomoya-namekawa/tf-file-organize/commit/fba920425de94310d04133db08f4b5c35589abca))
* improve RawBody processing for complete idempotency ([5d4c186](https://github.com/tomoya-namekawa/tf-file-organize/commit/5d4c186213d51d12e5653bb3f753cd9240872267))
* remove unused validatePath function and update test expectations ([96e158f](https://github.com/tomoya-namekawa/tf-file-organize/commit/96e158f3dd0f67fecbdb5ab173612e43cb255ac4))
* resolve file removal issue when blocks are reorganized by config ([1761e27](https://github.com/tomoya-namekawa/tf-file-organize/commit/1761e276b22413e2926256c6cf2f7da384329ae8))
* resolve idempotency issue with RawBody processing ([a1e71bd](https://github.com/tomoya-namekawa/tf-file-organize/commit/a1e71bddb61369cdf5818ced33ac091bf78c9096))

## [0.5.0](https://github.com/tomoya-namekawa/tf-file-organize/compare/v0.4.0...v0.5.0) (2025-06-22)


### ⚠ BREAKING CHANGES

* The repository name has changed. Users need to update their import paths and installation commands.

### Features

* enhance configuration system and test data ([9f2c78b](https://github.com/tomoya-namekawa/tf-file-organize/commit/9f2c78b17e079724cd3b415a2a6008857bb2bf1e))
* implement idempotent file organization with backup functionality ([4ac733b](https://github.com/tomoya-namekawa/tf-file-organize/commit/4ac733bd60cc69b2f9a6c412511fd65383cd003c))
* migrate to subcommand architecture and update documentation ([3b726b8](https://github.com/tomoya-namekawa/tf-file-organize/commit/3b726b83625d8331f280f119ad1b39db8c614a82))


### Bug Fixes

* correct case4 test input file for multi-cloud scenario ([bed9e66](https://github.com/tomoya-namekawa/tf-file-organize/commit/bed9e6666eb46f4ea9bffa2d9b0ee489f177aab2))
* update CI workflow to use plan subcommand ([14614db](https://github.com/tomoya-namekawa/tf-file-organize/commit/14614dbe162f3339bf1243a0687cb77c4d9d5677))


### Code Refactoring

* rename repository from terraform-file-organize to tf-file-organize ([7b50908](https://github.com/tomoya-namekawa/tf-file-organize/commit/7b50908ca0c2bbe7ceedfb3f85116f5939227019))

## [0.4.0](https://github.com/tomoya-namekawa/tf-file-organize/compare/v0.3.0...v0.4.0) (2025-06-22)


### Features

* remove addComments feature and add recursive flag with validation ([2b14c64](https://github.com/tomoya-namekawa/tf-file-organize/commit/2b14c643149933201c8f32b43e202626be9fe3ca))

## [0.3.0](https://github.com/tomoya-namekawa/tf-file-organize/compare/v0.2.0...v0.3.0) (2025-06-22)


### Features

* replace gopkg.in/yaml.v3 with github.com/goccy/go-yaml ([f4783d3](https://github.com/tomoya-namekawa/tf-file-organize/commit/f4783d378daee297956e60b619c21a2cb5fa2455))


### Bug Fixes

* **deps:** update module github.com/zclconf/go-cty to v1.16.3 ([#11](https://github.com/tomoya-namekawa/tf-file-organize/issues/11)) ([091ca1a](https://github.com/tomoya-namekawa/tf-file-organize/commit/091ca1a8a05138bac5c37f3e24e7d2c024391198))

## [0.2.0](https://github.com/tomoya-namekawa/tf-file-organize/compare/v0.1.1...v0.2.0) (2025-06-22)


### Features

* improve version detection for go install compatibility ([4be5dde](https://github.com/tomoya-namekawa/tf-file-organize/commit/4be5dde80c200a575e65d90bfc60c2a4b35ed872))

## [0.1.1](https://github.com/tomoya-namekawa/tf-file-organize/compare/v0.1.0...v0.1.1) (2025-06-22)


### Bug Fixes

* resolve GoReleaser deprecated warnings and errors ([8a35123](https://github.com/tomoya-namekawa/tf-file-organize/commit/8a35123c00b1e59553df4b517f47d60f0c39fc06))

## 0.1.0 (2025-06-22)
