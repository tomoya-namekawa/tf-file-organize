# Changelog

## 1.0.0 (2025-06-22)


### Features

* add case5 test for template expressions with nested blocks ([5b36cfc](https://github.com/tomoya-namekawa/terraform-file-organize/commit/5b36cfc2c1e53e6ac68093ee2203b5b66445adab))
* add GitHub Actions linting with actionlint and zizmor ([d6d6411](https://github.com/tomoya-namekawa/terraform-file-organize/commit/d6d64118455dcabff10a032699d85ad0ed8e6081))
* implement automated release system with release-please ([a2c4cf6](https://github.com/tomoya-namekawa/terraform-file-organize/commit/a2c4cf68e232bfdabb060a71f8516efca97c0ecb))
* implement case4 complex nested blocks and template expressions test ([875336d](https://github.com/tomoya-namekawa/terraform-file-organize/commit/875336dfbf28d91f8e00c3dc4b37e14ab68b08d6))
* implement clean architecture with usecase layer and stable output ([9af41ab](https://github.com/tomoya-namekawa/terraform-file-organize/commit/9af41aba46302108292f29a76bedd03f84b472d7))
* implement terraform file organization CLI tool ([72266e0](https://github.com/tomoya-namekawa/terraform-file-organize/commit/72266e0962fde01d6259b406ad40c5d9575dbf59))
* migrate CI to use mise for tool management ([ea0d6dd](https://github.com/tomoya-namekawa/terraform-file-organize/commit/ea0d6dd79eae2fbfc35d48f39444b8acc730ac78))
* remove gosec and pin mise-action for CI stability ([67ce543](https://github.com/tomoya-namekawa/terraform-file-organize/commit/67ce5434d7070bf5ca4a5352b1d82d92e954f04c))
* separate workflow linting into dedicated workflow ([37f2a3b](https://github.com/tomoya-namekawa/terraform-file-organize/commit/37f2a3bb6eabe6dd8bddf6dc0c07a2b5007b84c7))
* update golangci-lint to v2.1.6 in CI workflow ([3976979](https://github.com/tomoya-namekawa/terraform-file-organize/commit/3976979422251bb9127859a27df23f3c078b39a5))
* upgrade to golangci-lint v2 and modernize codebase ([17d5ad5](https://github.com/tomoya-namekawa/terraform-file-organize/commit/17d5ad53eb867a7f724042eee22be526b8107c63))
* use golangci-lint-action in CI instead of mise ([6304c8c](https://github.com/tomoya-namekawa/terraform-file-organize/commit/6304c8c1ff4802f7e8fd5b3012864c41d7286b4b))


### Bug Fixes

* add issues write permission for release-please labeling ([161efe5](https://github.com/tomoya-namekawa/terraform-file-organize/commit/161efe5ee63fcb0a40df6a9155163e0b67e2340c))
* correct golangci-lint configuration and gosec installation ([537d840](https://github.com/tomoya-namekawa/terraform-file-organize/commit/537d840628d9f788d8fb0bbd5b4ba8819221c08e))
* correct golangci-lint output configuration ([e574c95](https://github.com/tomoya-namekawa/terraform-file-organize/commit/e574c95fdcbe5ea6d03bb1d804dff8f6fc1a6d74))
* correct release-please workflow configuration ([fbb653d](https://github.com/tomoya-namekawa/terraform-file-organize/commit/fbb653d53e1f2bfe5ebc8a613343409d2866f016))
* format import statements to satisfy goimports linter ([025469b](https://github.com/tomoya-namekawa/terraform-file-organize/commit/025469b8eb6dea97c25114c100c5b1dd9d58ae0b))
* improve template expression handling in buildTemplateTokens ([9ff71f2](https://github.com/tomoya-namekawa/terraform-file-organize/commit/9ff71f2cdb6cc2d167856bfbf05d9c06e4912882))
* remove duplicate golden file test and correct module path ([ecdfe56](https://github.com/tomoya-namekawa/terraform-file-organize/commit/ecdfe566ac88ed63d32add14f5223d0162df3588))
* resolve all lint errors and warnings ([bd0a40d](https://github.com/tomoya-namekawa/terraform-file-organize/commit/bd0a40d0752c7eedaa7bc325bb13baee59537c2d))
* resolve CI failures with mise tool configuration ([8a68c9a](https://github.com/tomoya-namekawa/terraform-file-organize/commit/8a68c9a236c4b24910dfcd12985fc259b8ba402d))
* resolve duplicate resource names in test data and improve HCL writer ([7ee93cd](https://github.com/tomoya-namekawa/terraform-file-organize/commit/7ee93cd313ad1ea5bb8c813adbb7bf422217856b))
* resolve TestGoldenFiles/case2 failure ([97493fc](https://github.com/tomoya-namekawa/terraform-file-organize/commit/97493fc036d6e949a0f21fee6f7188c0e9603c59))
* update actions/cache to v4.1.2 to resolve CI failures ([f5179a2](https://github.com/tomoya-namekawa/terraform-file-organize/commit/f5179a2c5f74e91828202f8a880a69ba71aa40f3))
* update actions/cache to v4.2.3 (latest stable) ([4e0fdbf](https://github.com/tomoya-namekawa/terraform-file-organize/commit/4e0fdbfdc425a794e72f8f63bb4e4d34de88d2a0))
* update module path to correct GitHub repository URL ([a07368a](https://github.com/tomoya-namekawa/terraform-file-organize/commit/a07368a5b51754bf3f987095b5ab460ecb35cc90))
