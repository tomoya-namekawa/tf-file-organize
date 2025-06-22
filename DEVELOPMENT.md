# 開発ガイド

terraform-file-organizeの開発に関する詳細なガイドです。

## 開発環境のセットアップ

### 必要な依存関係

- Go 1.24.4+
- github.com/hashicorp/hcl/v2
- github.com/spf13/cobra
- golangci-lint (lintingのため)

### 初期セットアップ

```bash
# リポジトリのクローン
git clone https://github.com/tomoya-namekawa/terraform-file-organize.git
cd terraform-file-organize

# 依存関係のインストール
go mod tidy

# ビルド
go build -o terraform-file-organize
```

## 開発コマンド

### ビルドとテスト

```bash
# プロジェクトビルド
go build -o terraform-file-organize

# 全テストの実行
go test ./...

# パッケージ別テスト
go test ./internal/config
go test ./internal/parser
go test ./internal/splitter  
go test ./internal/writer
go test ./internal/usecase

# 統合テスト
go test -v ./integration_test.go
go test -v ./integration_golden_test.go

# 単一テストの実行
go test -run TestGroupBlocks ./internal/splitter

# ゴールデンファイルテスト（回帰検出のため重要）
go test -run TestGoldenFiles -v

# カバレッジ付きテスト
go test -v -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### Linting

```bash
# golangci-lintの実行
golangci-lint run

# CI用のlintコマンド（タイムアウト設定付き）
golangci-lint run --timeout=5m
```

### 開発時のテスト実行

```bash
# テスト実行（単一ファイル）
./terraform-file-organize testdata/terraform/sample.tf --dry-run

# テスト実行（ディレクトリ）
./terraform-file-organize testdata/terraform --output-dir tmp/test --dry-run

# 設定ファイル付きテスト
./terraform-file-organize testdata/terraform --config testdata/configs/terraform-file-organize.yaml --dry-run
```

## アーキテクチャと開発ガイドライン

### コード構造

本ツールはクリーンアーキテクチャの原則に従って設計されており、以下のレイヤーで構成されています：

- **CLI層** (`cmd/`): コマンドライン引数の解析
- **ユースケース層** (`internal/usecase/`): ビジネスロジックの調整とセキュリティ検証
- **ドメイン層** (`internal/`): コア機能（パーサー、スプリッター、ライター、設定）
- **データ層** (`pkg/types/`): データ構造の定義

### 開発時の重要な原則

#### 1. 決定的出力の維持

CI/CDやバージョン管理との互換性を確保するため、出力は常に決定的である必要があります：

- **リソースの並び順**: グループ内でアルファベット順にソート
- **属性の並び順**: HCL属性をアルファベット順にソート
- **ファイル名の並び順**: 出力ファイル名をアルファベット順にソート
- **フォーマット**: `hclwrite.Format`による一貫したフォーマット

#### 2. セキュリティファースト

- すべてのファイルパス操作は `filepath.Clean` と `filepath.Base` を使用
- 入力値の検証をusecase層で実装
- パストラバーサル攻撃対策を徹底

#### 3. HCL処理の統一

- 必ずHashiCorp公式のhcl/v2ライブラリを使用
- HCL構文の処理は公式パーサーに依存

### テスト戦略

#### 単体テスト

- 全ての `internal/` パッケージに対応する `*_test.go` ファイル
- インポートサイクル回避のため別テストパッケージ使用（例：`config_test`）

#### 統合テスト

- `integration_test.go`: バイナリベースのCLIテスト
- `integration_golden_test.go`: **重要** - ゴールデンファイルテスト

#### ゴールデンファイルテスト

**最も重要なテスト**です。回帰を防ぐため実際の出力と期待値を厳密に比較します。

```bash
# ゴールデンファイルテストの実行
go test -run TestGoldenFiles -v

# 期待値ファイルの更新（出力形式変更時）
./terraform-file-organize testdata/integration/case1/input -o testdata/integration/case1/expected
./terraform-file-organize testdata/integration/case2/input -o testdata/integration/case2/expected
./terraform-file-organize testdata/integration/case3/input -o testdata/integration/case3/expected
```

#### テストデータ構造

```
testdata/
├── terraform/          # 基本的なサンプルファイル
├── configs/            # 設定ファイルの例
└── integration/        # ゴールデンファイルテストケース
    ├── case1/          # 基本ブロック（デフォルト設定）
    ├── case2/          # 複数ファイルの基本グループ化
    └── case3/          # 設定ファイルによるカスタムグループ化
```

## CI/CD

### GitHub Actions

プロジェクトは包括的なCI/CDパイプラインを持っています：

- **pinact-check**: GitHub Actionsのハッシュ固定確認
- **test**: テストスイートの実行（レース検出・カバレッジ付き）
- **lint**: golangci-lintによる静的解析
- **build**: バイナリビルドと動作確認
- **security**: gosecによるセキュリティスキャン

### セキュリティ機能

- 全てのGitHub ActionsはコミットSHAハッシュで固定
- pinactによる自動検証
- gosecセキュリティスキャナー統合
- renovatebotによる依存関係管理

## 貢献時の注意点

### プルリクエスト前のチェックリスト

1. **全テストの実行**
   ```bash
   go test ./...
   ```

2. **ゴールデンファイルテストの実行**
   ```bash
   go test -run TestGoldenFiles -v
   ```

3. **Linting**
   ```bash
   golangci-lint run
   ```

4. **出力形式を変更した場合**
   - ゴールデンファイルの期待値を更新
   - 決定的出力が維持されているか確認

### よくある問題と解決方法

#### 1. ゴールデンファイルテストの失敗

出力形式を変更した場合、期待値ファイルの更新が必要です：

```bash
# 新しい出力で期待値を更新
./terraform-file-organize testdata/integration/case1/input -o testdata/integration/case1/expected
```

#### 2. インポートサイクル

テストファイルは別パッケージ（例：`package config_test`）として作成してください。

#### 3. 非決定的出力

コレクション（スライス、マップ）はソートしてから出力してください。

## デバッグとトラブルシューティング

### ログ出力

```bash
# 詳細なテスト出力
go test -v ./...

# 特定のテストのデバッグ
go test -run TestSpecificFunction -v ./internal/package
```

### 手動テスト

```bash
# ドライランでの動作確認
./terraform-file-organize testdata/terraform/sample.tf --dry-run

# 実際のファイル作成
mkdir -p tmp/manual-test
./terraform-file-organize testdata/terraform/sample.tf -o tmp/manual-test
```

### 設定ファイルのテスト

```bash
# カスタム設定でのテスト
./terraform-file-organize testdata/terraform --config testdata/configs/terraform-file-organize.yaml -o tmp/config-test --dry-run
```

## 開発フローとリリースプロセス

### Conventional Commitsによる開発フロー

このプロジェクトはConventional Commitsとrelease-pleaseによる自動リリースを採用しています。

#### コミットメッセージの形式

```bash
# 新機能の追加
git commit -m "feat: add support for terraform modules"

# バグ修正
git commit -m "fix: resolve parsing error for nested blocks"

# パフォーマンス改善
git commit -m "perf: optimize HCL parsing performance"

# リファクタリング
git commit -m "refactor: simplify resource grouping logic"

# ドキュメント更新
git commit -m "docs: update installation instructions"

# テストの追加
git commit -m "test: add golden file tests for case4"

# CI/CDの変更
git commit -m "ci: update GitHub Actions workflow"

# ビルドシステムの変更
git commit -m "build: update go.mod dependencies"

# その他の変更
git commit -m "chore: update development documentation"
```

#### バージョンに与える影響

- `feat:` → **minor** バージョンアップ (0.1.0 → 0.2.0)
- `fix:` → **patch** バージョンアップ (0.1.0 → 0.1.1)
- `perf:`, `refactor:`, `docs:`, `test:`, `ci:`, `build:`, `chore:` → **patch** バージョンアップ
- `BREAKING CHANGE:` フッター → **major** バージョンアップ (0.1.0 → 1.0.0)

#### 自動リリースプロセス

1. **コミット**: Conventional Commitsでmainブランチにコミット
2. **PR作成**: release-pleaseが自動でversion bump PRを作成
3. **リリース**: PRをマージするとGoReleaserが自動実行
4. **成果物**: GitHub Releasesにバイナリとchangelogが公開

#### ブランチ戦略

- **main**: 安定版ブランチ（ここへのマージでリリースPRが作成される）
- **feature branches**: 機能開発用ブランチ
- **release-please--branches--main--components--terraform-file-organize**: release-pleaseが自動作成するリリースPR用ブランチ

### リリース設定ファイル

- `.release-please-manifest.json`: 現在のバージョン管理
- `release-please-config.json`: リリース設定
- `.goreleaser.yaml`: バイナリビルド設定

### 手動操作が必要な場合

通常は自動リリースを使用しますが、以下の場合は手動操作が必要です：

#### リリースPRのマージ

release-pleaseが作成するPRは手動でレビュー・マージする必要があります：

1. release-pleaseがPR作成（`chore(main): release v1.0.0`）
2. PRの内容確認（CHANGELOG、バージョン更新）
3. PRのマージ
4. 自動でGoReleaserが実行され、リリース完了

#### 緊急リリース

緊急時は以下の手順でリリース可能：

```bash
# 1. 修正をコミット
git commit -m "fix: critical security issue"

# 2. release-pleaseを手動実行（GitHub Actions）
# または以下でローカル実行
npm install -g release-please
release-please release-pr --repo-url=https://github.com/tomoya-namekawa/terraform-file-organize
```

## 技術仕様

### アーキテクチャ

本ツールはクリーンアーキテクチャの原則に従って設計されており、以下のレイヤーで構成されています：

- **CLI層** (`cmd/`): コマンドライン引数の解析
- **ユースケース層** (`internal/usecase/`): ビジネスロジックの調整とセキュリティ検証
- **ドメイン層** (`internal/`): コア機能（パーサー、スプリッター、ライター、設定）
- **データ層** (`pkg/types/`): データ構造の定義

### 安定した出力の保証

CI/CDやバージョン管理との互換性を確保するため、以下の仕組みで決定的な出力を実現：

- **リソースの並び順**: グループ内でアルファベット順にソート
- **属性の並び順**: HCL属性をアルファベット順にソート
- **ファイル名の並び順**: 出力ファイル名をアルファベット順にソート
- **フォーマット**: `hclwrite.Format`による一貫したフォーマット

### セキュリティ機能

- **パストラバーサル対策**: `filepath.Clean`と`filepath.Base`による安全なパス処理
- **入力検証**: 不正なファイルパスや設定値の検証
- **シンボリックリンク対策**: 危険なシンボリックリンクのスキップ

### バージョン管理システム

#### Go Install対応

Go 1.18以降の`runtime/debug.BuildInfo`を活用して、以下の優先順位でバージョンを検出：

1. **GoReleaser**: ldflags注入されたバージョン（最優先）
2. **go install @version**: モジュールバージョンから自動検出
3. **開発ビルド**: VCS revisionから短縮ハッシュを生成

これにより、`go install`でも正しいバージョンが表示されます。

## 貢献ガイドライン

### プルリクエストとイシュー

プルリクエストやイシューの報告を歓迎します。バグ報告や機能要求は GitHub Issues をご利用ください。

### 貢献する前に

1. **Issue確認**: 既存のIssueを確認し、重複を避けてください
2. **フォーク**: リポジトリをフォークして作業用ブランチを作成
3. **テスト**: 変更後は必ず全テストを実行
4. **コミット**: Conventional Commitsの規約に従ってコミット
5. **プルリクエスト**: 明確な説明とともにPRを作成

### コード品質基準

- **テストカバレッジ**: 60%以上を維持
- **Linting**: golangci-lintによるチェックをパス
- **ゴールデンファイル**: 出力変更時は期待値ファイルを更新
- **セキュリティ**: gosecによるセキュリティチェックをパス

このガイドに従って開発を進めることで、品質とセキュリティを保ちながら機能を拡張できます。