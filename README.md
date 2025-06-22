# terraform-file-organize

Terraformファイルをリソースタイプごとに分割・整理するGoのCLIツールです。

## 概要

大きなTerraformファイルを管理しやすい小さなファイルに分割し、リソースタイプごとに整理します。各ブロックタイプは特定の命名規則に従って別々のファイルに配置されます。

## 特徴

- 🔧 **完全なTerraformサポート**: すべてのTerraformブロックタイプ（resource、data、variable、output、provider、terraform、locals、module）に対応
- 📁 **スマートなファイル分割**: リソースタイプごとに論理的にファイルを分割
- 📂 **ディレクトリ処理**: 単一ファイルまたはディレクトリ全体の一括処理に対応
- ⚙️ **カスタム設定**: YAML設定ファイルによる柔軟なグループ化ルール
- 🎯 **直感的な命名規則**: 分かりやすいファイル命名パターン
- 👀 **ドライランモード**: 実際のファイル作成前にプレビュー可能
- ⚡ **高速処理**: HashiCorp公式のHCLパーサーを使用
- 🔒 **安定した出力**: 決定的な出力でCI/CDやバージョン管理に最適
- 🛡️ **セキュリティ対策**: パストラバーサル攻撃対策など包括的なセキュリティ機能
- ✅ **包括的テスト**: ゴールデンファイルテストによる回帰防止

## インストール

### go installを使用

```bash
# 最新版をインストール
go install github.com/tomoya-namekawa/terraform-file-organize@latest

# 特定のバージョンをインストール
go install github.com/tomoya-namekawa/terraform-file-organize@v1.0.0
```

### バイナリのダウンロード

[GitHub Releases](https://github.com/tomoya-namekawa/terraform-file-organize/releases) から各プラットフォーム用のバイナリをダウンロードできます。

### ソースからビルド

```bash
git clone https://github.com/tomoya-namekawa/terraform-file-organize.git
cd terraform-file-organize
go build -o terraform-file-organize
```

## 使用方法

### 基本的な使用例

```bash
# 単一ファイルをドライランでプレビュー
terraform-file-organize main.tf --dry-run

# ディレクトリ内の全Terraformファイルを整理
terraform-file-organize ./terraform-configs

# カスタム出力ディレクトリを指定
terraform-file-organize main.tf --output-dir ./organized

# 設定ファイルを使用してカスタムグループ化
terraform-file-organize . --config terraform-file-organize.yaml
```

### オプション

- `<input-path>`: 入力Terraformファイルまたはディレクトリ（必須・位置引数）
- `-o, --output-dir`: 出力ディレクトリ（デフォルト: 入力パスと同じ）
- `-c, --config`: 設定ファイルパス（デフォルト: 自動検出）
- `-d, --dry-run`: ドライランモード（実際のファイル作成なし）

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

## 使用例

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

## 設定ファイル

### 自動検出

ツールは以下の順序で設定ファイルを自動検出します：

1. `terraform-file-organize.yaml`
2. `terraform-file-organize.yml`
3. `.terraform-file-organize.yaml`
4. `.terraform-file-organize.yml`

### 設定例

```yaml
# terraform-file-organize.yaml
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

# デフォルトファイル名を変更
overrides:
  variable: "vars.tf"
  locals: "common.tf"

# 除外パターン（個別ファイルのまま）
exclude:
  - "aws_instance_special*"
```

### 設定のメリット

- **論理的なグループ化**: 関連リソースを意味のあるファイルにまとめる
- **ワイルドカードサポート**: `aws_s3_*` で S3 関連リソースを一括指定
- **柔軟な命名**: プロジェクトに合わせたファイル名のカスタマイズ
- **選択的除外**: 特定のリソースは個別ファイルのまま保持

## 開発

開発に関する詳細な情報は [DEVELOPMENT.md](DEVELOPMENT.md) を参照してください。

## 実用例

### プロジェクト整理の例

大きな `main.tf` を整理する場合：

```bash
# 現在のプロジェクト
terraform-file-organize . --dry-run
# → ファイル分割の結果をプレビュー

terraform-file-organize .
# → 実際に分割実行（同じディレクトリ内）
```

### CI/CDでの活用

```bash
# 設定ファイルをプロジェクトに配置
echo "
groups:
  - name: infrastructure
    filename: infrastructure.tf
    patterns: [aws_vpc, aws_subnet*, aws_security_group*]
  - name: compute
    filename: compute.tf  
    patterns: [aws_instance, aws_launch_*, aws_autoscaling_*]
" > terraform-file-organize.yaml

# 自動整理（設定ファイル自動検出）
terraform-file-organize .
```

## 開発・貢献

開発に関する詳細情報、技術仕様、貢献方法については [DEVELOPMENT.md](DEVELOPMENT.md) を参照してください。

## ライセンス

このプロジェクトは MIT ライセンスの下で公開されています。