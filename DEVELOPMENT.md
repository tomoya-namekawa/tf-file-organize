# 開発ガイド

tf-file-organizeの開発に関する詳細なガイドです。

## 開発環境のセットアップ

### ツール管理

このプロジェクトは[mise](https://mise.jdx.dev/)を使用して開発ツールを統一管理しています。

```bash
# miseをインストール（初回のみ）
curl https://mise.run | sh

# プロジェクトの依存ツールをインストール
mise install

# ツールバージョンの確認
mise list
```

### 初期セットアップ

```bash
# リポジトリのクローン
git clone https://github.com/tomoya-namekawa/tf-file-organize.git
cd tf-file-organize

# 依存関係のインストール
go mod tidy

# ビルド
go build -o tf-file-organize
```

## 開発コマンド

### ビルドとテスト

```bash
# プロジェクトビルド
go build -o tf-file-organize

# 全テストの実行
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# 重要：ゴールデンファイルテスト（回帰検出）
go test -run TestGoldenFiles -v

# CLI機能テスト（全サブコマンド）
go test -run TestCLI -v

# パッケージ別テスト
go test ./internal/config -v
go test ./internal/parser -v
go test ./internal/splitter -v
go test ./internal/writer -v
go test ./internal/usecase -v

# カバレッジ目標: 60%以上
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### コード品質チェック

```bash
# Linting
golangci-lint run

# フォーマット
go mod tidy

# ワークフロー検証
actionlint
```

### 開発時のテスト実行

```bash
# プレビューモード
./tf-file-organize plan testdata/terraform/sample.tf

# ディレクトリ処理
./tf-file-organize plan testdata/terraform

# 設定ファイル付きテスト
./tf-file-organize plan testdata/terraform --config testdata/configs/tf-file-organize.yaml

# 設定ファイル検証
./tf-file-organize validate-config testdata/configs/tf-file-organize.yaml
```

## アーキテクチャ

### コード構造

本ツールはクリーンアーキテクチャの原則に従って設計されており、以下のレイヤーで構成されています：

- **CLI層** (`cmd/`): サブコマンド定義と引数解析
- **ユースケース層** (`internal/usecase/`): ビジネスロジックの調整とセキュリティ検証
- **ドメイン層** (`internal/`): コア機能（パーサー、スプリッター、ライター、設定）
- **データ層** (`pkg/types/`): データ構造の定義

### サブコマンド構造

- `run <input-path>`: ファイル整理の実行
- `plan <input-path>`: プレビューモード（旧 --dry-run）
- `validate-config <config-file>`: 設定ファイル検証
- `version`: バージョン情報表示

## 開発時の重要な原則

### 1. 冪等性の維持

ツールは複数回実行しても一貫した結果を保証する必要があります：

- **デフォルト動作**: ソースファイルを削除して重複を防止
- **バックアップオプション**: `--backup`でソースファイルを'backup'ディレクトリに移動
- **スマート競合解決**: 設定ルールを考慮したファイル削除ロジック

### 2. 決定的出力の維持

CI/CDやバージョン管理との互換性を確保するため、出力は常に決定的である必要があります：

- **リソースの並び順**: グループ内でアルファベット順にソート
- **属性の並び順**: HCL属性をアルファベット順にソート
- **ファイル名の並び順**: 出力ファイル名をアルファベット順にソート

### 3. コメント保持

ブロック内コメントは常に保持される必要があります：

- **デュアルパースィング**: 標準HCL + `hclsyntax`パーシング
- **RawBody抽出**: コメントを含む元ソース抽出
- **Raw Block再構築**: 元コンテンツでの出力

### 4. セキュリティファースト

- すべてのファイルパス操作は `filepath.Clean` と `filepath.Base` を使用
- 入力値の検証をusecase層で実装
- パストラバーサル攻撃対策を徹底

### 5. パターンマッチング

複雑なパターンマッチングシステムをサポート：

- **シンプルパターン**: `aws_s3_*`
- **サブタイプパターン**: `resource.aws_instance.web*`
- **ブロックタイプパターン**: `variable`, `output.debug_*`
- **複数ワイルドカード**: `*special*`

## テスト戦略

### テスト構造

- **単体テスト**: 全ての `internal/` パッケージに対応、別テストパッケージ使用
- **CLIテスト** (`cli_test.go`): バイナリ実行による機能テスト
- **ゴールデンファイルテスト** (`golden_test.go`): **最重要** - 回帰検出

### テストデータ構造

```
testdata/
├── terraform/          # 基本的なサンプルファイル
├── configs/            # 設定ファイルの例
└── integration/        # ゴールデンファイルテストケース
    ├── case1/          # 基本ブロック（デフォルト設定）
    ├── case2/          # 複数ファイルの基本グループ化
    ├── case3/          # 設定ファイルによるカスタムグループ化
    ├── case4/          # 複雑なマルチクラウド構成（25ブロック）
    └── case5/          # テンプレート式とネストブロック
```

### ゴールデンファイルテスト

**最も重要なテスト**です。出力変更時は期待値ファイルの更新が必要：

```bash
# ゴールデンファイルテストの実行
go test -run TestGoldenFiles -v

# 期待値ファイルの更新（出力形式変更時）
cp tmp/integration-test/case*/* testdata/integration/case*/expected/
```

## 開発ワークフロー

### プレコミットチェックリスト

```bash
# 1. フォーマットとLint
golangci-lint run

# 2. カバレッジ付きテスト
go test -v -coverprofile=coverage.out ./...

# 3. ゴールデンファイル検証
go test -run TestGoldenFiles -v

# 4. ビルドと統合テスト
go build -o tf-file-organize
./tf-file-organize plan testdata/terraform/sample.tf

# 5. ワークフロー検証
actionlint
```

### 出力変更時の注意点

出力形式を変更した場合：

1. **ゴールデンファイルの期待値を更新**
2. **決定的出力が維持されているか確認**
3. **コメント保持機能が正常に動作するか確認**

## CI/CD設定

### GitHub Actions

- **Main CI Pipeline** (`.github/workflows/ci.yml`): test, lint, build with security scanning
- **Workflow Lint Pipeline** (`.github/workflows/workflow-lint.yml`): actionlint and pinact checks

### セキュリティ機能

- GitHub Actionsのコミットハッシュ固定
- gosecセキュリティスキャン統合
- パストラバーサル攻撃対策

### Tool Versions (mise管理)

```toml
[tools]
go = "latest"
golangci-lint = "v2.1.6"
actionlint = "latest"
pinact = "latest"
"npm:@goreleaser/goreleaser" = "latest"
```

## リリースプロセス

### Conventional Commitsによる自動リリース

このプロジェクトはConventional Commitsとrelease-pleaseによる自動リリースを採用しています。

#### コミットメッセージの形式

```bash
# 新機能の追加
git commit -m "feat: add plan subcommand for preview mode"

# バグ修正
git commit -m "fix: resolve pattern matching for complex wildcards"

# パフォーマンス改善
git commit -m "perf: optimize HCL parsing performance"

# リファクタリング
git commit -m "refactor: simplify resource grouping logic"

# ドキュメント更新
git commit -m "docs: update README for subcommand structure"

# テストの追加・修正
git commit -m "test: add golden file tests for idempotency"

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
- `BREAKING CHANGE:` フッター → **major** バージョンアップ (0.1.0 → 1.0.0)
- その他 → **patch** バージョンアップ

#### 自動リリースプロセス

1. **コミット**: Conventional Commitsでmainブランチにコミット
2. **PR作成**: release-pleaseが自動でversion bump PRを作成
3. **リリース**: PRをマージするとGoReleaserが自動実行
4. **成果物**: GitHub Releasesにバイナリとchangelogが公開

### 設定ファイル

- `.release-please-manifest.json`: 現在のバージョン管理
- `release-please-config.json`: リリース設定
- `.goreleaser.yaml`: バイナリビルド設定

### 手動リリーステスト

```bash
# ローカルでリリースビルドをテスト
make release-snapshot

# GoReleaser設定の確認
make release-check
```

## よくある問題と解決方法

### 1. ゴールデンファイルテストの失敗

出力形式を変更した場合、期待値ファイルの更新が必要です：

```bash
# 新しい出力で期待値を更新
go test -run TestGoldenFiles -v
cp tmp/integration-test/case*/* testdata/integration/case*/expected/
```

### 2. インポートサイクル

テストファイルは別パッケージ（例：`package config_test`）として作成してください。

### 3. 非決定的出力

コレクション（スライス、マップ）はソートしてから出力してください。

### 4. パターンマッチングのデバッグ

```bash
# 特定のパターンマッチングテスト
go test ./internal/splitter -run TestGroupBlocksWithConfig -v

# 設定ファイル検証
./tf-file-organize validate-config testdata/configs/tf-file-organize.yaml
```

## 貢献ガイドライン

### プルリクエスト前のチェック

1. **全テストの実行とパス**
2. **ゴールデンファイルテスト実行**
3. **golangci-lint実行**
4. **Conventional Commitsでのコミット**
5. **actionlint実行（ワークフロー変更時）**

### コード品質基準

- **テストカバレッジ**: 60%以上を維持
- **Linting**: golangci-lintによるチェックをパス
- **ゴールデンファイル**: 出力変更時は期待値ファイルを更新
- **セキュリティ**: gosecによるセキュリティチェックをパス

このガイドに従って開発を進めることで、品質とセキュリティを保ちながら機能を拡張できます。