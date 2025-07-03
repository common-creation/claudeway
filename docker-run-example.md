# claudeway Docker実行例

## 手動でのDocker実行方法

claudewayコンテナを手動で起動する場合、以下のようなdocker runコマンドを使用します：

```bash
# イメージをビルド
docker build -t claudeway:latest .

# コンテナを起動
docker run -it --rm \
  --name claudeway-$(echo -n "$(pwd)" | sha256sum | cut -c1-8) \
  -v "$(pwd):$(pwd)" \
  -w "$(pwd)" \
  -e HOST_UID=$(id -u) \
  -e HOST_GID=$(id -g) \
  -e HOST_USER=$USER \
  --env-file <(env | grep -v '^_') \
  claudeway:latest \
  /bin/bash
```

## 環境変数の説明

- `HOST_UID`: ホストユーザーのUID
- `HOST_GID`: ホストユーザーのGID  
- `HOST_USER`: ホストユーザー名

これらの環境変数により、コンテナ内でホストと同じユーザーが作成され、ファイルのパーミッション問題を回避できます。

## entrypoint.shの動作

1. コンテナ起動時、最初はrootユーザーとして実行されます
2. `HOST_UID`、`HOST_GID`、`HOST_USER`が設定されている場合：
   - 指定されたUID/GIDでユーザーとグループを作成
   - sudoersにパスワードなしでsudo実行できる設定を追加
   - `sudo -u $HOST_USER -E`でスクリプトを再実行
3. ホストユーザーとして実行される際：
   - `CLAUDEWAY_COPY`で指定されたファイルをコピー
   - `CLAUDEWAY_INIT`で指定された初期化コマンドを実行
   - 最後に指定されたコマンド（デフォルトは`/bin/bash`）を実行

## 追加のバインドマウント

追加のディレクトリをマウントする場合：

```bash
docker run -it --rm \
  --name claudeway-$(echo -n "$(pwd)" | sha256sum | cut -c1-8) \
  -v "$(pwd):$(pwd)" \
  -v "/opt/bin:/opt/bin" \
  -v "$HOME/.gitconfig:/host$HOME/.gitconfig:ro" \
  -w "$(pwd)" \
  -e HOST_UID=$(id -u) \
  -e HOST_GID=$(id -g) \
  -e HOST_USER=$USER \
  -e CLAUDEWAY_COPY="~/.gitconfig" \
  -e CLAUDEWAY_INIT="npm ci;go mod download" \
  claudeway:latest \
  /bin/bash
```