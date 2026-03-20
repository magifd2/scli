# scli — 設計概要

## 1. 目的

`scli` は **ユーザーとして**（ボットではなく）動作するコマンドラインSlackクライアントです。
GUIクライアントへのコンテキストスイッチをなくし、ターミナルから直接Slackの投稿・閲覧ができることを目的とします。

---

## 2. 設計方針

- **セキュリティファースト** — トークンはデフォルトでOSキーチェーンに保管。平文フォールバックはオプトイン。
- **小さく・集中** — 各コマンドは1つのことだけを行う。バックグラウンドデーモン不要。
- **関心の分離** — CLIレイヤー・APIレイヤー・認証レイヤー・設定レイヤー・出力レイヤーは独立。
- **テスト可能な設計** — 全レイヤーはインターフェース経由で接続。I/Oとビジネスロジックは混在させない。

---

## 3. システム全体像

```
ユーザー（ターミナル）
     │  scli <コマンド>
     ▼
┌─────────────────┐
│   scli (CLI)    │  ← cobraベースのコマンドディスパッチャ
└────────┬────────┘
         │
  ┌──────┴──────┐
  │  認証レイヤー │  ← トークン解決（環境変数 → 設定 → キーチェーン）
  └──────┬──────┘
         │
  ┌──────┴──────┐
  │ Slack API   │  ← Slack Web API（HTTPS）
  │  クライアント │
  └──────┴──────┘
         │
  ┌──────┴──────┐
  │  出力レイヤー │  ← カラー整形テキスト（デフォルト）またはJSON（--json）
  └─────────────┘
```

---

## 4. レイヤー詳細

### 4.1 cmd/

Cobraによるコマンド定義。各サブコマンドは即座に内部サービスに委譲し、ビジネスロジックは持たない。

```
cmd/
  root.go          # グローバルフラグ: --workspace, --json, --no-color
  auth.go          # auth login / logout / list
  workspace.go     # workspace list / use
  channel.go       # channel list / read / post
  dm.go            # dm list / send / read
  unread.go        # unread
  search.go        # search
  user.go          # user list（DM宛先解決用）
```

### 4.2 internal/auth/

Slack の OAuth 2.0 PKCE フロー。
責務：
- OAuthコールバックを受け取るためのローカルHTTPサーバーを起動。
- 認証コードをユーザートークン（`xoxp-`）に交換。
- トークンをキーチェーン/設定レイヤーに渡して保存。

### 4.3 internal/config/

トークンとワークスペースプロファイルの管理。

**トークン解決の優先順位（ワークスペースごと）：**

```
1. 環境変数   SLACK_TOKEN_<WORKSPACE>（またはデフォルト用 SLACK_TOKEN）
2. .env ファイル（プロジェクトローカルまたは ~/.config/scli/.env）
3. 設定ファイル  ~/.config/scli/config.json
4. OSキーチェーン（internal/keychain/ 参照）
```

設定ファイルスキーマ（`config.json`）：

```json
{
  "default_workspace": "myteam",
  "workspaces": {
    "myteam": {
      "token": "",           // 空の場合はキーチェーンを使用
      "team_id": "T012AB3C4",
      "user_id": "U012AB3C4"
    }
  }
}
```

### 4.4 internal/keychain/

OSのシークレットストレージへの薄い抽象化レイヤー：

| プラットフォーム | バックエンド |
|----------------|------------|
| macOS | Keychain（Security.framework経由） |
| Linux | libsecret / `secret-tool` |
| Windows | Windows Credential Manager |

Goライブラリ: [`zalando/go-keyring`](https://github.com/zalando/go-keyring)

インターフェース：

```go
type Store interface {
    Get(workspace string) (token string, err error)
    Set(workspace string, token string) error
    Delete(workspace string) error
}
```

### 4.5 internal/slack/

Slack Web APIクライアント。各メソッドが1つのAPIエンドポイントに対応。

必要なOAuthスコープ（ユーザートークン）：

| スコープ | 用途 |
|---------|------|
| `channels:read` | パブリックチャンネル一覧 |
| `groups:read` | プライベートチャンネル一覧 |
| `im:read` | DM一覧 |
| `im:write` | DMチャンネルを開く |
| `mpim:read` | グループDM一覧 |
| `channels:history` | パブリックチャンネルのメッセージ読み取り |
| `groups:history` | プライベートチャンネルのメッセージ読み取り |
| `im:history` | DMメッセージ読み取り |
| `mpim:history` | グループDMメッセージ読み取り |
| `chat:write` | メッセージ投稿 |
| `files:write` | ファイルアップロード |
| `search:read` | メッセージ検索 |
| `users:read` | ユーザー名解決 |

### 4.6 internal/output/

標準出力へのレンダリング。

- デフォルト：ANSIカラー整形、人間が読みやすい形式。
- `--json` フラグ：生JSON（`jq` へのパイプに適する）。
- `--no-color` フラグ：プレーンテキスト（非TTY環境向け）。

TTYを自動検出し、stdoutがターミナルでない場合はカラーを自動無効化。

---

## 5. コマンドリファレンス

### 認証

```
scli auth login  [--workspace <name>]   ブラウザでOAuth認証し、トークンを保存
scli auth logout [--workspace <name>]   トークンを削除
scli auth list                          認証済みワークスペースを一覧表示
```

### ワークスペース

```
scli workspace list                     設定済みワークスペースを一覧表示
scli workspace use <name>               デフォルトワークスペースを変更
```

### チャンネル

```
scli channel list                       参加中のチャンネルを一覧表示
scli channel read <channel>             最近のメッセージを読む
  [--limit N]                           取得件数（デフォルト: 20）
  [--unread]                            未読メッセージのみ表示
  [--thread <timestamp>]                指定スレッドを表示
scli post <channel> <message>           メッセージを投稿
  [--file <path>]                       ファイルを添付
  [--thread <timestamp>]                スレッドに返信
```

### DM

```
scli dm list                            DM会話の一覧表示
scli dm read <user>                     ユーザーとのDMを読む
  [--limit N]
scli dm send <user> <message>           DMを送信
  [--thread <timestamp>]
```

### 未読

```
scli unread                             全チャンネル・DMの未読件数を表示
  [--workspace <name>]
```

### 検索

```
scli search <query>                     メッセージを検索
  [--in <channel>]                      チャンネルを絞り込む
  [--limit N]
```

### ユーザー

```
scli user list                          ワークスペースメンバーを一覧表示（DM宛先解決用）
```

---

## 6. チャンネル / ユーザーの解決

コマンドが `<channel>` または `<user>` を受け取る際の解決ロジック：

- `C`, `G`, `D`, `U` で始まる場合（Slack ID）→ そのまま使用。
- `#` で始まる場合 → `#` を除いてチャンネル名で検索。
- `@` で始まる場合 → `@` を除いてユーザー名/表示名で検索。
- それ以外 → 名前で検索を試みる。曖昧または見つからない場合はエラー。

---

## 7. ディレクトリ構成（予定）

```
scli/
  cmd/                  CLIエントリポイント（cobra）
  internal/
    auth/               OAuthフロー
    config/             設定・トークン解決
    keychain/           OSキーチェーン抽象化
    slack/              Slack APIクライアント
    output/             フォーマッタ（カラー / JSON）
  docs/
    design/             設計ドキュメント（英語）
    ja/                 日本語翻訳
  scripts/
    hooks/              Gitフック（pre-commit, pre-push）
  Makefile
  go.mod
  go.sum
  CHANGELOG.md
  CLAUDE.md
```

---

## 8. エラー処理

- APIエラーはSlackエラーコードと人間が読めるメッセージで表示。
- ネットワークエラーは接続確認またはトークン有効性の確認を促す。
- すべてのエラーは非ゼロのステータスコードで終了（スクリプティングに適する）。

---

## 9. v1スコープ外

- リアクション
- リアルタイムイベントストリーミング（WebSocket / Events API）
- メッセージ編集・削除
- スラッシュコマンド
- インタラクティブコンポーネント（モーダル、ボタン等）

---

*原文（英語）: `docs/design/overview.md`*
