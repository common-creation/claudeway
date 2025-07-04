<p align="center">
  <img width="320" src="./assets/claudeway_logo.png" />
</p>
<h2 align="center">
  claudeway - The Ultimate Defense: Securing Claude Code Execution
</h2>

----

Claude CodeをはじめとするAIコーディングエージェントを安全にDockerコンテナ内で実行するためのCLIツールです。

[English version available](README.en.md)

## 仕組み

カレントディレクトリをDockerコンテナにbind mountして起動します。  
このとき、一部を除きホストの環境変数を引き継ぎ（dotenvフレンドリー）、設定ファイルに応じてコンテナを初期化、ホストと同一のUID・GIDでexecします。  
これにより、万が一AIコーディングエージェントが `rm -rf --no-preserve-root` を実行したとしても、ホストに致命的な影響がないようにします。

## インストール

```bash
go install github.com/mohemohe/claudeway@latest
```

## 使い方

### 初期設定

```bash
# グローバル設定とDockerアセットを初期化
claudeway init --global

# プロジェクトごとの設定を初期化
claudeway init
```

### 基本的な使い方

```bash
# コンテナを起動して対話的シェルに入る
claudeway up

# すでに起動しているコンテナの対話的シェルに入る
claudeway exec

# コンテナを停止・削除
claudeway down
```

### その他のコマンド

```bash
# Dockerイメージをビルド
claudeway image build

# キャッシュなしでイメージをビルド
claudeway image build --no-cache
```

## 設定ファイル

`claudeway.yaml` の形式：

```yaml
# ボリュームマウント設定
bind:
  - /var/run/docker.sock:/var/run/docker.sock  # Dockerソケット
  - ~/.claude.json:~/.claude.json               # Claude設定
  - ~/.claude:~/.claude                         # Claudeディレクトリ

# コンテナ内にコピーするファイル
copy:
  - ~/.gitconfig                                 # Git設定
  - ~/.ssh                                       # SSH鍵

# 初期化コマンド（コンテナ起動時に実行）
init:
  - curl -Ls get.docker.com | sh                # Docker インストール
  - asdf plugin add nodejs                       # Node.js プラグイン追加
  - asdf install nodejs 22.17.0                  # Node.js インストール
  - asdf global nodejs 22.17.0                   # デフォルトバージョン設定
  - npm i -g @anthropic-ai/claude-code          # Claude Code インストール
```

実際の例は [`claudeway.yaml`](./claudeway.yaml) を確認してください。
