# tf-file-organize

Terraformファイルをリソースタイプごとに分割・整理するGoのCLIツールです。

## 概要

大きなTerraformファイルを管理しやすい小さなファイルに分割し、リソースタイプごとに整理します。各ブロックタイプは特定の命名規則に従って別々のファイルに配置されます。

## 特徴

- 🔧 **完全なTerraformサポート**: すべてのTerraformブロックタイプ（resource、data、variable、output、provider、terraform、locals、module）に対応
- 📁 **スマートなファイル分割**: リソースタイプごとに論理的にファイルを分割
- 📂 **ディレクトリ処理**: 単一ファイルまたはディレクトリ全体の一括処理に対応
- ⚙️ **複雑なパターンマッチング**: サブタイプパターン（`resource.aws_instance.web*`）を含む高度なグループ化
- 🎯 **直感的な命名規則**: 分かりやすいファイル命名パターン
- 👀 **プレビューモード**: `plan`コマンドで実際のファイル作成前にプレビュー可能
- 💾 **バックアップ機能**: `--backup`オプションで元ファイルを安全に保管
- ⚡ **高速処理**: HashiCorp公式のHCLパーサーを使用
- 🔒 **冪等性**: 複数回実行しても一貫した結果を保証
- 💬 **コメント保持**: ブロック内コメントを完全保持
- 🛡️ **セキュリティ対策**: パストラバーサル攻撃対策など包括的なセキュリティ機能
- ✅ **包括的テスト**: ゴールデンファイルテストによる回帰防止

## インストール

### go installを使用

```bash
# 最新版をインストール
go install github.com/tomoya-namekawa/tf-file-organize@latest

# 特定のバージョンをインストール
go install github.com/tomoya-namekawa/tf-file-organize@v1.0.0
```

### バイナリのダウンロード

[GitHub Releases](https://github.com/tomoya-namekawa/tf-file-organize/releases) から各プラットフォーム用のバイナリをダウンロードできます。

### ソースからビルド

```bash
git clone https://github.com/tomoya-namekawa/tf-file-organize.git
cd tf-file-organize
go build -o tf-file-organize
```

## 使用方法

### 基本コマンド

```bash
# プレビューモード（実際のファイル作成なし）
tf-file-organize plan main.tf

# ファイルを実際に整理（元ファイルは削除）
tf-file-organize run main.tf

# バックアップ付きで整理（元ファイルはbackupディレクトリに移動）
tf-file-organize run main.tf --backup

# ディレクトリ全体を整理
tf-file-organize run ./terraform-configs

# カスタム出力ディレクトリを指定
tf-file-organize run main.tf --output-dir ./organized

# 設定ファイルを使用してカスタムグループ化
tf-file-organize run . --config tf-file-organize.yaml

# 設定ファイルの検証
tf-file-organize validate-config tf-file-organize.yaml
```

### サブコマンド

| コマンド | 説明 | 使用例 |
|---------|------|--------|
| `run` | ファイルを実際に整理・作成 | `tf-file-organize run main.tf` |
| `plan` | プレビューモード（dry-run） | `tf-file-organize plan main.tf` |
| `validate-config` | 設定ファイルの検証 | `tf-file-organize validate-config config.yaml` |
| `version` | バージョン情報を表示 | `tf-file-organize version` |

### オプション

#### runコマンド
- `<input-path>`: 入力Terraformファイルまたはディレクトリ（必須・位置引数）
- `-o, --output-dir`: 出力ディレクトリ（デフォルト: 入力パスと同じ）
- `-c, --config`: 設定ファイルパス（デフォルト: 自動検出）
- `-r, --recursive`: ディレクトリを再帰的に処理
- `--backup`: 元ファイルをbackupディレクトリに移動

#### planコマンド
- 同じオプション（`--backup`を除く）

## ファイル命名規則

| ブロックタイプ | 命名規則 | 例 |
|---------------|----------|-----|
| resource | `resource__{resource_type}.tf` | `resource__aws_instance.tf` |
| data | `data__{data_source_type}.tf` | `data__aws_ami.tf` |
| variable | `variables.tf` | `variables.tf` |
| output | `outputs.tf` | `outputs.tf` |
| provider | `providers.tf` | `providers.tf` |
| terraform | `terraform.tf` | `terraform.tf` |
| locals | `locals.tf` | `locals.tf` |
| module | `module__{module_name}.tf` | `module__vpc.tf` |

## 設定ファイル

### 自動検出

ツールは以下の順序で設定ファイルを自動検出します：

1. `tf-file-organize.yaml`
2. `tf-file-organize.yml`
3. `.tf-file-organize.yaml`
4. `.tf-file-organize.yml`

### 設定例

```yaml
# tf-file-organize.yaml
groups:
  # AWSネットワーク関連をまとめる
  - name: "network"
    filename: "network.tf"
    patterns:
      - "aws_vpc"
      - "aws_subnet"
      - "aws_security_group*"

  # AWSコンピュート関連をまとめる
  - name: "compute"
    filename: "compute.tf"
    patterns:
      - "aws_instance"
      - "aws_lb*"

  # サブタイプパターンの使用例
  - name: "web_infrastructure"
    filename: "web-infra.tf"
    patterns:
      - "resource.aws_instance.web*"
      - "resource.aws_lb.web*"
      - "resource.aws_security_group.web"

  # 変数とoutputのカスタマイズ
  - name: "variables"
    filename: "vars.tf"
    patterns:
      - "variable"

  - name: "debug_outputs"
    filename: "debug-outputs.tf"
    patterns:
      - "output.debug_*"

# ファイル名パターンで除外（個別ファイルのまま）
exclude_files:
  - "*special*.tf"
  - "debug-*.tf"
```

### パターンマッチング機能

- **シンプルパターン**: `aws_s3_*`でS3関連リソースを一括指定
- **サブタイプパターン**: `resource.aws_instance.web*`で特定のリソース名パターンを指定
- **ブロックタイプパターン**: `variable`、`output.debug_*`でブロックタイプを指定
- **複数ワイルドカード**: `*special*`のように複数の`*`を使用可能

## 実用例

### 入力ファイル（main.tf）

```hcl
terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}

resource "aws_instance" "web" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = var.instance_type
  
  tags = {
    Name = "web-server"
  }
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"]
  
  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*"]
  }
}
```

### 出力結果

分割後、以下のファイルが作成されます：

- `terraform.tf` - terraform設定ブロック
- `providers.tf` - providerブロック
- `variables.tf` - variableブロック
- `resource__aws_instance.tf` - aws_instanceリソース
- `data__aws_ami.tf` - aws_amiデータソース

### プロジェクト整理の例

```bash
# 現在のプロジェクトをプレビュー
tf-file-organize plan .
# → ファイル分割の結果をプレビュー

# バックアップ付きで実際に分割実行
tf-file-organize run . --backup
# → 元ファイルはbackup/に移動、整理されたファイルを作成

# 設定ファイル付きで整理
tf-file-organize run . --config my-config.yaml
```

### CI/CDでの活用

```bash
# 設定ファイルをプロジェクトに配置
cat > tf-file-organize.yaml << 'EOF'
groups:
  - name: infrastructure
    filename: infrastructure.tf
    patterns: 
      - aws_vpc
      - aws_subnet*
      - aws_security_group*
  - name: compute
    filename: compute.tf
    patterns:
      - aws_instance
      - aws_launch_*
      - aws_autoscaling_*
exclude_files:
  - "*temp*.tf"
EOF

# 設定の検証
tf-file-organize validate-config tf-file-organize.yaml

# 自動整理（設定ファイル自動検出）
tf-file-organize run .
```

## 冪等性とファイル管理

このツールは冪等的な動作を保証し、複数回実行しても一貫した結果を提供します：

- **デフォルト動作**: 整理後、元ファイルは自動削除（重複を防ぐため）
- **バックアップオプション**: `--backup`フラグで元ファイルをbackupディレクトリに保存
- **スマート競合解決**: 設定を考慮した重複ファイル検出と削除

## 開発・貢献

開発に関する詳細情報、技術仕様、貢献方法については [DEVELOPMENT.md](DEVELOPMENT.md) を参照してください。

## ライセンス

このプロジェクトは MIT ライセンスの下で公開されています。