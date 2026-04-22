# sample-api

バックエンド API のサンプル実装です。

## ローカル起動

1. 初回だけ env を作成

```bash
cp .env.local.example .env.local
```

2. 開発用 MySQL を起動して初期化

```bash
make docker-up
make db-setup
```

3. API を起動

```bash
make run
```

- API: `http://localhost:8080`
- MySQL: `localhost:3306`

## Docker 起動

Front / API / DB をまとめて起動する場合は、リポジトリルートで次を使う。

```bash
make up
make down
```

Docker 側の API はコンテナ起動時に migration と seed を自動実行する。
