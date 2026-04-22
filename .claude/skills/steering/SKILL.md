---
name: steering
description: >
  `docs/steering/` をプロジェクトメモリとして管理するスキル。
  steering ファイルが空または未作成の場合は Bootstrap（初回生成）、
  既存ファイルがある場合は Sync（コードとの乖離検出・更新）を実行する。
  「steering を更新して」「steering を生成して」「ドキュメントを同期して」などのトリガーで起動する。
allowed-tools: Bash, Read, Write, Edit, Glob, Grep
---

# Steering Management

`docs/steering/` をプロジェクトの永続的なメモリとして維持する。

**ミッション**:
- Bootstrap: コードベースからコア steering を初回生成する
- Sync: steering とコードベースの乖離を検出・修正する
- Preserve: ユーザーのカスタマイズは不可侵。更新は追記型で行う

**成功基準**:
- steering はパターン・原則を記述し、網羅的なリストにしない
- コードとの乖離を検出・報告する
- `docs/steering/*.md` はすべて（コアファイル＋カスタムファイル）同等に扱う

---

## シナリオ判定

`docs/steering/` の状態を確認する:

- **Bootstrap モード**: ディレクトリが空、または `product.md` / `tech.md` / `structure.md` のいずれかが欠けている
- **Sync モード**: 3 つのコアファイルがすべて存在する

---

## Bootstrap フロー

1. `docs/settings/templates/steering/` からテンプレートを読み込む（存在する場合）
2. コードベースを JIT で解析する:
   - `Glob` でソースファイルを検索する
   - `Read` で README、go.mod、設定ファイルなどを確認する
   - `Grep` でパターンを探す
3. 以下のパターンを抽出する（リストではなくパターンを）:
   - **product.md**: 目的・提供価値・コア機能
   - **tech.md**: フレームワーク・設計決定・規約
   - **structure.md**: ディレクトリ構成・命名規則・インポートルール
4. steering ファイルを生成する
5. `docs/settings/rules/steering-principles.md` が存在すれば読み込んでレビューに活用する
6. 生成内容のサマリーを報告する

**焦点**: 判断を導くパターンを記述する。ファイルや依存ライブラリの網羅的なカタログは避ける。

---

## Sync フロー

1. 既存の steering ファイルをすべて読み込む（`docs/steering/*.md`）
2. コードベースの変更を JIT で解析する
3. 乖離を検出する:
   - **Steering → Code**: steering に書かれた要素がコードに存在しない → Warning
   - **Code → Steering**: コードに新パターンがある → 更新候補
   - **カスタムファイル**: 関連性を確認する
4. 更新案を提示する（追記型、ユーザーセクションを保持）
5. 変更点・警告・推奨事項を報告する

**更新方針**: 追記する。置き換えない。ユーザーが書いたセクションを保持する。

---

## 粒度の原則

> 「新しいコードが既存パターンに従っているなら、steering の更新は不要なはず。」

パターンと原則を記述する。網羅的なリストは避ける。

**悪い例**: ディレクトリツリーのすべてのファイルを列挙する  
**良い例**: 構成パターンを例示で説明する

---

## 注意事項

- エージェント固有のツールディレクトリ（`.cursor/`、`.gemini/`、`.claude/` など）は steering に記載しない
- `docs/settings/` の内容は steering に記載しない（設定はメタデータであり、プロジェクト知識ではない）
- `docs/specs/` と `docs/steering/` への軽い言及は許容。それ以外の `docs/` 配下は避ける
- シークレット・パスワード・APIキーは絶対に含めない

---

## 完了報告フォーマット

### Bootstrap 完了時:

```
✅ Steering 生成完了

## 生成ファイル:
- product.md: [概要]
- tech.md: [技術スタック]
- structure.md: [構成パターン]

レビューして Source of Truth として承認してください。
```

### Sync 完了時:

```
✅ Steering 更新完了

## 変更内容:
- tech.md: [変更点]
- structure.md: [変更点]

## コード乖離:
- [乖離が検出された場合に記載]

## 推奨事項:
- [追加推奨ファイルなど]
```
