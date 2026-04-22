---
name: go-clean-arch
description: >
  go-clean-arch プロジェクトの current-state な Clean Architecture パターンに従って
  コード生成・修正・レビューを行う。新しいドメイン（Entity / Repository / Service / Handler）
  の追加、既存ドメインへの機能追加、既存レイヤーの修正、DI 配線、テスト追加、エラーフロー確認、
  mock の作成・更新方針の確認で使う。新機能追加時は必ず対象ドメインの既存実装とテストを先に読み、
  最も近い既存機能を写経元にして最小差分で実装する。既存コードの整列・legacy drift の是正でも使う。
  DB schema 変更は repo ルートの `db/migrate/` に置き、seed データは `db/seed/` に置く。business layer には置かない。
---

# go-clean-arch ガイド

> **前提**: このスキルは現在のリポジトリ実装を観察起点にするが、最終的な整列先はこの `SKILL.md` に書かれた repo-wide ルールとする。

---

## 1. ソースオブトゥルース

判断に迷ったら、以下の優先順位で従う。

1. **ユーザーが今回求めている目的**（新機能追加 / 既存コード整列 / レビュー）
2. **この `SKILL.md` の MUST ルールと repo-wide invariants**
3. **対象パッケージの既存コード**
4. `references/` のサンプル

`references/` はテンプレートであり、仕様書ではない。対象ファイルの近傍コードと矛盾したら、
近傍コードのパターンを優先する。ただし、近傍コードが `SKILL.md` の MUST ルールや
repo-wide invariants と矛盾するなら、その既存コードは **legacy drift** とみなし、
コピー元ではなく整列対象として扱う。

### 1-1. Repo-Wide Invariants

次は「新規コードだけでなく既存コードも寄せる対象」として扱う。

- IF は消費側で宣言する
- `domain/` に外部依存を入れない
- `internal/rest` の共有シンボルは再定義しない
- handler まで返す domain error は status map と test を追随させる
- error 握り潰し、resource close 漏れ、`rows.Err()` 未確認を放置しない
- 実装変更に対して mock と test を同じ変更セットで追随させる
- DB migration は repo ルートの `db/migrate/` に置く（seed は `db/seed/` に分離する）

### 1-2. 新機能追加時の基本姿勢

新機能を作るときは、次の順で判断する。

1. **先に読む**: 対象ドメインの service / handler / repository / test / mock を読む
2. **写経元を決める**: 最も近い既存メソッドや endpoint を 1 つ決める
3. **既存ファイルへ足す**: 新しい package / helper / file を増やす前に既存ファイルへ追加できるか確認する
4. **最小差分で閉じる**: 必要な層だけ触り、横断リファクタはしない
5. **欠陥は複製しない**: 構成や命名は合わせるが、既存のバグやテスト不足まではコピーしない

この 5 点は、実装・修正・レビューのどれでも優先する。

### 1-3. 既存コード整列時の基本姿勢

既存コードをスキル準拠に直すときは、次の順で進める。

1. **drift を特定する**: どの MUST ルールや repo-wide invariant から外れているか言語化する
2. **最小の整列単位を決める**: file 単位、package 単位、または 1 endpoint / 1 usecase 単位に閉じる
3. **近傍コードを観察する**: 命名や責務分割は近傍に合わせる
4. **規約へ寄せる**: IF 配置、error handling、安全性、test 追随は `SKILL.md` 側へ寄せる
5. **波及を制御する**: 大規模な一括整列は避け、必要なら複数の小変更に分ける

---

## 2. コア構造

Clean Architecture の中心ルールは変わらない。

```text
rest/handler → service/usecase → domain ← repository adapter
```

代表的な配置は次の通り。

```text
domain/                    ← エンティティ + センチネルエラー
{domain}/                  ← Service（UseCase）+ Repository Interface
  mocks/                   ← テスト用 mock（手動保守）
  service.go
  service_test.go
internal/
  repository/mysql/        ← DB 実装
  rest/                    ← HTTP ハンドラ + Service Interface
  rest/mocks/              ← handler/service テスト用 mock（手動保守）
db/
  migrate/               ← DB schema migration（golang-migrate, .up.sql のみ）
  seed/                  ← 初期データ（DML のみ、スキーマ変更は含めない）
app/main.go                ← DI 配線・サーバー起動
```

---

## 3. 新規実装・既存整列の標準フロー

新しく機能を追加するときも、既存コードを整列するときも、いきなり実装しない。次の順番を崩さない。

### 3-1. 着手前に必ず読む

最低でも次を確認する。

1. 同じドメインの service 実装
2. 同じドメインの handler 実装
3. 同じドメインの repository 実装
4. それぞれのテスト
5. interface が変わるなら対応する mock

新しいドメインを追加する場合でも、まずは最も近い既存ドメインの一式を読む。
このリポジトリでは通常 `article` が最初の参照先になる。

### 3-2. 写経元を 1 つ決める

追加する機能に最も近い既存実装を 1 つ決め、その形を優先して踏襲する。

- メソッド名
- 引数順
- error の返し方
- JSON レスポンス形式
- SQL の書き方
- テストの置き方
- mock の作り方

複数の既存実装があるときは、**同じファイル内**または**同じパッケージ内**の近い実装を優先する。
README や references より、近傍コードの一貫性を優先する。
ただし、写経元が repo-wide invariant に反している箇所はそのまま複製しない。

### 3-3. 追加先を決める

- **既存ドメインの機能追加**なら、まず同じドメインの既存ファイルを拡張する
- **新しいドメイン追加**が必要なときだけ、新しい `{domain}/` と対応する `internal/` 配下を作る
- **既存コード整列**なら、まず同じ責務の既存ファイルを修正し、新しい package や abstraction は増やさない
- **DB schema 変更**なら、migration file は repo ルートの `db/migrate/` に置く（seed データは `db/seed/` に分離する）
- 迷ったら「新しい package を増やさず、既存 package の既存ファイルに足せるか」を先に検討する
- 共通化のためだけの helper / util / 抽象 interface は追加しない。ただし、同一ロジックが同パッケージ内で完全に重複している（引数・戻り値・処理がほぼ同一で名前だけ異なる関数群など）場合は、同パッケージ内の専用ファイル（例: `internal/rest/params.go`）への集約を許容する

### 3-4. 編集順を固定する

1. `domain/`: entity / error が必要なときだけ追加する
2. `{domain}/service.go`: usecase と Repository IF を追加・変更する
3. `internal/repository/mysql/{domain}.go`: 永続化実装を合わせる
4. `internal/rest/{domain}.go`: handler / request / response / Service IF を合わせる
5. 各テストと mock を同じ変更セットで更新する
6. `app/main.go`: 新しい配線が必要なときだけ触る

interface 変更、handler 追加、repository 追加のどれであっても、**実装だけ先に入れて mock や test を後回しにしない**。
既存コード整列でも、**production code だけ直して test を据え置きにしない**。

### 3-5. 完了条件

新機能追加後または既存コード整列後は、少なくとも次を満たす。

- 追加・変更した interface と実装、mock が一致している
- 新しい error を返すなら handler のステータスマッピングとテストが追随している
- handler の入出力を変えたなら handler test が追随している
- SQL を変えたなら repository test が追随している
- 影響した package の `go test` を実行している
- 触った既存コードが repo-wide invariant から外れたまま残っていない

---

## 4. 依存ルール

| レイヤー | 依存してよいもの | 避けるもの |
|---|---|---|
| `domain/` | 標準ライブラリのみ | `github.com/` 以下の外部依存、他レイヤー |
| `{domain}/service.go` | `domain/`、標準ライブラリ、純粋な補助ライブラリ、Repository IF | `internal/rest`、`internal/repository/mysql`、DB/HTTP 実装詳細 |
| `internal/rest/` | Service IF、`domain/`、Echo、validator、rest 内共有シンボル | Service 実装への直接依存 |
| `internal/repository/mysql/` | `domain/`、`database/sql`、`internal/repository` の helper、必要な logger / util | Service / handler パッケージへの依存 |
| `app/main.go` | DI に必要な全レイヤー | ビジネスロジック |

ポイントは「依存は内側へ」だが、**service 層が pure な補助ライブラリを使うこと自体は許容**する。
このリポジトリでも `errgroup` や logger を使っている。禁止なのは、adapter 実装への逆依存。

---

## 5. インターフェース配置

インターフェースは**消費側で宣言**する。

```go
// {domain}/service.go
type FooRepository interface { ... }

// internal/rest/foo.go
type FooService interface { ... }
```

- Repository IF は service 側に置く
- Service IF は handler 側に置く
- mysql 実装側や service 実装側に「自分が実装するための IF」を置かない

---

## 6. 追加・修正時の判断順

**Step 1 — エンティティか**

- struct、value object、センチネルエラーは `domain/`
- `domain/` は外部依存を持たせない

**Step 2 — ユースケースか**

- ビジネスロジックは `{domain}/service.go`
- 必要な Repository IF もここで宣言する

**Step 3 — DB / 外部 I/O 実装か**

- `internal/repository/mysql/{domain}.go`
- 上位レイヤーの interface に合わせて実装する

**Step 4 — HTTP 入出力か**

- `internal/rest/{domain}.go`
- Service IF もここで宣言する

**Step 5 — 配線か**

- `app/main.go`
- Repository → Service → Handler の組み立てだけに留める

---

## 7. MUST ルール

### 7-1. 既存コードは観察起点であり、免罪符ではない

同じ責務の近傍ファイルに既存パターンがあるなら、まずそれを観察する。
ただし、その既存パターンが `SKILL.md` の MUST ルールや repo-wide invariants と矛盾するなら、
その箇所は「現状追認」ではなく整列対象として扱う。

### 7-2. 既存ドメインへの機能追加は、既存ファイルの拡張を優先する

たとえば既存の `article` に機能を足すなら、最初に検討すべき場所は次の既存ファイル。

- `article/service.go`
- `internal/rest/article.go`
- `internal/repository/mysql/article.go`
- それぞれのテスト

新しい helper package や別ファイルは、既存ファイルに追加すると不自然になる場合に限る。

### 7-3. `domain/` は外部依存ゼロ

`time` などの標準ライブラリは可。外部パッケージ import は禁止。

### 7-4. 共有済みシンボルを再定義しない

`internal/rest` では `ResponseError`、`getStatusCode`、`defaultNum` など
既存の共有シンボルを流用する。別名での重複定義は避ける。

### 7-5. 命名・シグネチャ・並び順は近傍コードに揃える

追加メソッドやテストは、同じファイル内の既存パターンに合わせる。

- `Fetch / GetByID / Store / Update / Delete` のような命名順
- `ctx` を先頭に置く引数順
- handler の `NewXHandler`, `GetByID`, `Store` のような名前
- test の `TestXxx` / `t.Run(...)` の粒度

小さな一貫性の崩れも積み上がるので、既存の並び順と書式を意識する。

### 7-6. mock は通常のソースとして手動保守する

`mocks/` 配下は生成物ではなく、通常の Go ソースとして扱う。
interface 変更時は mock も同じ変更セットで更新する。

方針:

- 必要なメソッドだけを実装した小さな mock を優先する
- テストごとに必要十分な振る舞いだけを持たせる
- 実装より複雑な mock を作らない
- 生成ツール前提のコメントや運用に依存しない

### 7-7. `app/main.go` を薄く保つ

設定読み込み、接続初期化、DI、サーバ起動に限定する。

### 7-7a. migration は `db/migrate/` に置き、seed は `db/seed/` に置く

DB schema 変更を入れるときは、migration file は repo ルートの `db/migrate/` に置く。
初期データや開発用データは `db/seed/` に置き、schema 変更（DDL）とデータ投入（DML）を混在させない。

**Migration（`db/migrate/`）**
- ツール: `golang-migrate`
- ファイル命名: `YYYYMMDDHHMMSS_{table_name}.up.sql`
  - 例: `20260403120000_create_groups.up.sql`
- 内容: DDL のみ（CREATE TABLE, ALTER TABLE, DROP TABLE など）
- 適用履歴は `schema_migrations` テーブルで自動管理
- Makefile:
  - `make db-migrate` — 未適用の `.up.sql` を順番に実行
  - `make db-reset` — 開発環境限定・DB を完全削除して `db-migrate` で再構築（`APP_ENV=development` が必要）
  - `make db-state` — 適用済み / 未適用のマイグレーション一覧を表示

**Seed（`db/seed/`）**
- ファイル命名: `seed.sql`（全テーブル分を FK 依存順にまとめる）
- 内容: DML のみ（INSERT, TRUNCATE など）
- Makefile: `make db-seed`（冪等に設計すること）

**置いてよい場所**

| 種別 | 場所 |
|---|---|
| Schema 変更 SQL | `db/migrate/` |
| Seed データ SQL | `db/seed/` |
| 実行コード・runner | `Makefile`, CI, Docker entrypoint |

**置かない場所**: `domain/`, `{domain}/service.go`, `internal/rest/`, `internal/repository/mysql/`

既存の `db/migrations/` の単発 SQL があっても、新規の schema 変更は `db/migrate/` へ、seed は `db/seed/` へ寄せる。

### 7-8. 新しい domain エラーを handler まで流すなら、ステータスマッピングも確認する

このリポジトリでは `domain/errors.go` にセンチネルエラーを集約しているが、
**全エラーが自動で HTTP にマップされるわけではない**。

新しいエラーを追加したり、既存の `ErrBadParamInput` のようなエラーを
handler まで返す設計にする場合は、対象 handler の `getStatusCode` とテストも合わせて確認する。

### 7-9. REST は必要な操作だけ公開すればよい

service に存在しても、REST endpoint まで必ず公開する必要はない。

### 7-10. 不要な抽象化・横断リファクタを持ち込まない

新機能追加のついでに次を行わない。

- 共通化のためだけの helper 追加
- package 分割のやり直し
- 既存メソッド名の一括改名
- テストスタイルの全面変更

機能要件に必要な最小差分で終える。

補足：同一ロジックの完全重複（引数・戻り値・処理が同一で名前だけ異なる関数群）の除去は、上記の「不要な抽象化」には含まない。この場合は §3-3 の例外条件に従う。

### 7-11. 既存の欠陥まで複製しない

近傍コードの構成・責務分割・命名・response 形式・テスト粒度は踏襲する。
ただし次のようなものは「既存ルール」ではなく、改善余地として扱う。

- error の握り潰し
- resource close 漏れ
- `rows.Err()` や status map の見落とし
- テスト不足
- N+1 クエリ（ループ内の単件 SELECT）→ `IN` 句や `COUNT(DISTINCT id)` の 1 クエリへの置き換えを検討する
- 一覧 API の `total` カウントが検索フィルタ（`q` 等）を反映していない → COUNT クエリにも同条件を適用する
- DB 固有エラーの文字列マッチ判定（`strings.Contains(err.Error(), "Error 1062")` 等）→ `errors.As` + エラー番号による型安全な判定に置き換える

迷ったら、**API と構成は既存に合わせ、安全性と検証は強くする**。

### 7-12. 触った legacy code は同じ責務の範囲で整列する

既存コードを触るとき、同じ責務の範囲で次の drift が見えているなら、同じ変更で寄せることを優先する。

- status map の不足
- error 伝搬の欠落
- resource lifecycle の漏れ
- mock / test の追随漏れ
- IF 配置のズレ

ただし、変更が広く波及するなら一括で広げず、小さく分ける。

---

## 8. エラーハンドリング

このリポジトリで安定しているパターンは次の通り。

### 8-1. リクエスト入力エラー

- path / query の解析失敗: handler 側で直接返す
- `Bind` 失敗: 生の `err.Error()`
- バリデーション失敗: 生の `err.Error()`

### 8-2. Service / Repository 由来のエラー

- `ResponseError{Message: err.Error()}` でラップして返す

### 8-3. 現在の代表的なマッピング

```text
domain.ErrNotFound            → 404
domain.ErrConflict            → 409
domain.ErrInternalServerError → 500
その他                         → 実装側の getStatusCode に従う
```

補足:

- `domain.ErrBadParamInput` は domain に存在する
- ただし handler 側で必ず 4xx にマップされているとは限らない
- クライアント入力エラーとして見せたい場合は、対象 handler の `getStatusCode` を明示的に更新する

### 8-4. DB エラーの型安全な検出

MySQL などの DB 固有エラーを判定するときは、文字列マッチを使わず `errors.As` でドライバのエラー型に変換してエラー番号で判定する。

```go
// ✅ 型安全
var mysqlErr *mysql.MySQLError
if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
    return domain.ErrConflict
}

// ❌ 脆弱（ドライバのメッセージ変更で壊れる）
if strings.Contains(err.Error(), "Error 1062") { ... }
```

文字列マッチによるエラー判定は §7-11 の「改善余地」として整列対象にする。

---

## 9. データ・実装上の慣例

### 9-1. 時刻型

- 新しい timestamp フィールドは `time.Time` を優先する
- 既存エンティティが `string` を使っている場合は、移行タスクでない限り安易に型変更しない

### 9-2. カーソル

- cursor pagination は `internal/repository` の `EncodeCursor` / `DecodeCursor` を使う
- 独自の Base64 / time format 実装を増やさない

### 9-3. バリデーション関数

- ドメインごとに閉じた helper 名にする
- 既存の `isRequestValid` を他ドメインで使い回さない

### 9-4. 関連データ取得

- 関連 entity を複数件引くなら `errgroup + goroutine + channel` は有力な選択肢
- ただし、すべてのユースケースで必須ではない。既存サービスの複雑さと利得に合わせる

### 9-5. Repository のクエリ実装

- `QueryContext` 直呼びと `PrepareContext` ベースの両方が現行コードにある
- まずは**同じファイル・同じパッケージの既存パターン**に合わせる
- `PrepareContext` を使う場合も、resource close や error check は省略しない

ここでは近傍コードとの一貫性を優先するが、repo-wide invariants や安全性チェックはさらに優先する。

### 9-6. ID・ページネーション引数の型

| 用途 | 型 |
|---|---|
| Entity の ID（DB PRIMARY KEY） | `uint64` |
| `limit` / `offset`（ページネーション） | `int` |

- handler / service / repository を通じて型を一貫させる
- `int64` への変換が必要になる箇所は型ミスマッチのシグナルとして扱う
- `//nolint:gosec` が不可避になる設計は型の選択を見直す

---

## 10. 実装・整列時のチェックリスト

- 追加前に対象ドメインの service / handler / repository / test / mock を読んだか
- 写経元にする既存実装を 1 つ決めたか
- 追加先は既存ドメインか新規ドメインか、または既存コード整列か
- DB schema 変更（DDL）なら `db/migrate/` に `.up.sql` で置いているか
- Seed データ（DML）なら `db/seed/` に置き、`db/migrate/` と混在していないか
- service / handler / repository / test のどこまで触る必要があるか整理したか
- interface を変えたなら実装・mock・テストを同時に更新したか
- 既存の error / response / SQL / naming の形に揃っているか
- 既存の欠陥まで複製していないか
- repo-wide invariant に反する既存 drift を放置していないか
- 差分は機能要件に対して最小か

---

## 11. レビュー時のチェックリスト

- IF の宣言位置は消費側になっているか
- `domain/` に外部依存が入っていないか
- service が adapter 実装へ逆依存していないか
- handler のエラーレスポンス形式が既存パターンに揃っているか
- 新しい domain エラーを返すなら `getStatusCode` とテストが追随しているか
- cursor / time / validation helper の既存慣例に沿っているか
- `app/main.go` にユースケースが漏れていないか
- mock が interface と整合しているか
- mock が不要に肥大化していないか
- 既存コードの drift を「現状だから」で見逃していないか

---

## 12. Conditional References

必要なときだけ参照する。

- 新しいドメイン追加、Repository / Service / Handler 実装、DI 配線:
  `references/implementation.md`
- 既存コード整列、legacy cleanup、Repository / Service / Handler のテスト追加、mock 作成・更新:
  `references/implementation.md`
- Service / Repository / Handler のテスト追加、既存コード整列時の regression test、mock 作成・更新:
  `references/testing.md`
