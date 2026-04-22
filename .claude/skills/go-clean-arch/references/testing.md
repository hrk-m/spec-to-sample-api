# テストパターン

> このファイルもテンプレート。期待値の置き方は対象パッケージの既存テストを優先する。

## テスト追加前チェック

新機能のテストを書く前に、同じドメインの既存テストを読む。

- service の追加なら同じ package の service_test
- handler の追加なら同じ package の rest_test
- repository の追加なら同じ package の mysql_test

新しいテストスタイルを持ち込まず、最も近い既存ケースの構造を真似る。
実装を追加したら、同じ変更セットで影響層のテストも更新する。
既存コード整列でも、修正した drift を捕まえる regression test を同じ変更で足す。

## 基本方針

- テストパッケージは外部パッケージ (`package foo_test`, `package rest_test`, `package mysql_test`)
- mock は手動作成し、通常のソースとして保守する
- 成功系だけでなく、エラー系と境界条件も見る
- interface を変えたら mock とテストを同時に直す
- handler の status や response 形式を変えたら handler test で明示的に確認する
- 既存コード整列なら「何が直ったか」を落とし込むテストを最低 1 本は追加検討する

---

## 1. Mock の配置と作り方

- repository mock は `{domain}/mocks/` に置く
- service mock は `internal/rest/mocks/` に置く
- interface 全体を機械的に複製するより、実際のテストで使うメソッドを明示した小さな mock を優先する
- テストケースが少ない場合は、テストファイル内のローカル stub でもよい
- interface を変更したら、その変更セットで mock も直す

例:

```go
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/bxcodec/go-clean-arch/domain"
)

type FooRepository struct {
	mock.Mock
}

func (m *FooRepository) GetByID(ctx context.Context, id int64) (domain.Foo, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Foo), args.Error(1)
}
```

---

## 2. Service テスト（`foo/service_test.go`）

service テストでは、Repository mock を差し替えてユースケースの分岐を検証する。

```go
package foo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/bxcodec/go-clean-arch/foo"
	"github.com/bxcodec/go-clean-arch/foo/mocks"
)

func TestFooGetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := new(mocks.FooRepository)
		expected := domain.Foo{ID: 1, Name: "test"}

		mockRepo.On("GetByID", mock.Anything, int64(1)).Return(expected, nil).Once()

		svc := foo.NewService(mockRepo)
		result, err := svc.GetByID(context.TODO(), 1)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not-found", func(t *testing.T) {
		mockRepo := new(mocks.FooRepository)

		mockRepo.On("GetByID", mock.Anything, int64(1)).
			Return(domain.Foo{}, domain.ErrNotFound).Once()

		svc := foo.NewService(mockRepo)
		result, err := svc.GetByID(context.TODO(), 1)

		assert.Error(t, err)
		assert.Equal(t, domain.Foo{}, result)
		mockRepo.AssertExpectations(t)
	})
}
```

見るべき点:

- service が repository エラーをどう伝搬するか
- `Update` の時刻更新など、service 固有ロジックがあるか
- `Store` / `Delete` の事前存在確認や conflict 判定があるか
- 既存 service test の `t.Run` 粒度や assertion の置き方に揃っているか

---

## 3. Repository テスト（`internal/repository/mysql/foo_test.go`）

repository テストでは sqlmock で SQL と引数を検証する。

```go
package mysql_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/bxcodec/go-clean-arch/domain"
	mysqlRepo "github.com/bxcodec/go-clean-arch/internal/repository/mysql"
)

func TestStoreFoo(t *testing.T) {
	now := time.Now()
	item := &domain.Foo{
		Name:      "test",
		CreatedAt: now,
		UpdatedAt: now,
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected sqlmock error: %v", err)
	}

	query := "INSERT foo SET name=\\?, updated_at=\\?, created_at=\\?"
	prep := mock.ExpectPrepare(query)
	prep.ExpectExec().
		WithArgs(item.Name, item.UpdatedAt, item.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := mysqlRepo.NewFooRepository(db)
	err = repo.Store(context.TODO(), item)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), item.ID)
}
```

見るべき点:

- production code が `PrepareContext` を使うなら `ExpectPrepare` を使う
- `QueryContext` / `QueryRowContext` 直呼びなら `ExpectQuery` を使う
- cursor decode 失敗、not found、affected rows mismatch などの分岐も必要に応じて追加する
- SQL 文字列の比較方法も既存テストの粒度に揃える

---

## 4. Handler テスト（`internal/rest/foo_test.go`）

handler テストでは HTTP 入出力とエラーフォーマットを検証する。

```go
package rest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/bxcodec/go-clean-arch/internal/rest"
	"github.com/bxcodec/go-clean-arch/internal/rest/mocks"
)

func TestFooGetByID(t *testing.T) {
	mockSvc := new(mocks.FooService)
	mockSvc.On("GetByID", mock.Anything, int64(1)).
		Return(domain.Foo{ID: 1, Name: "test"}, nil).Once()

	e := echo.New()
	req, err := http.NewRequestWithContext(context.TODO(), echo.GET, "/foos/1", strings.NewReader(""))
	assert.NoError(t, err)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/foos/:id")
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(1))

	h := rest.FooHandler{Service: mockSvc}
	err = h.GetByID(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	mockSvc.AssertExpectations(t)
}
```

見るべき点:

- path / query parse error のステータス
- `Bind` / validation error のレスポンス形式
- `domain.ErrNotFound` / `domain.ErrConflict` / internal error のマッピング
- `Fetch` 系なら `X-Cursor` ヘッダ
- 既存 handler test と同じ request 組み立て方、path 設定、assertion 順を使う

---

## 5. 最低限のカバレッジ指針

新しい handler / service / repository を足すときは、少なくとも次を検討する。

- success path
- not found path
- conflict path があるならその path
- invalid input path
- repository / service から予期しない error が返る path

既存コードが軽めのテストでも、追加分では分岐の意味がある箇所を優先して押さえる。

機能追加では「実装だけ先に入れてテストは後回し」にしない。少なくとも影響した層の主要分岐は同じ変更で更新する。
既存コード整列でも同様で、status map 修正、error 伝搬修正、validation 修正のような挙動差分は test で固定する。

複数件操作（バルク INSERT・バルク存在チェック）がある場合は追加で検討する:

- 重複 ID を含む入力（deduplication が正しく動き、COUNT 比較がずれないか）
- 一部の ID が存在しない（`count != len(ids)` で `ErrNotFound` が返るか）
- 全 ID が存在しない
- repository エラー（`CountByIDs` / bulk INSERT が `ErrInternalServerError` を返す）
- 検索フィルタあり・なしで `total` の値が正しく変わるか（COUNT クエリとフィルタの整合）
