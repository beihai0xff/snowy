package redis

import (
	"context"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionStore_Get_Success(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewSessionStore(db)

	key := "session:sess-123:ctx"
	mock.ExpectGet(key).SetVal(`{"mode":"search"}`)
	mock.ExpectExpire(key, store.defaultTTL).SetVal(true)

	data, err := store.Get(context.Background(), "sess-123")

	require.NoError(t, err)
	assert.Equal(t, "search", data["mode"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionStore_Get_NotFound(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewSessionStore(db)

	mock.ExpectGet("session:missing:ctx").RedisNil()

	data, err := store.Get(context.Background(), "missing")

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionStore_Set_Success(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewSessionStore(db)

	key := "session:sess-123:ctx"
	mock.ExpectSet(key, []byte(`{"mode":"physics"}`), store.defaultTTL).SetVal("OK")

	err := store.Set(context.Background(), "sess-123", map[string]any{"mode": "physics"})

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionStore_Delete_Success(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewSessionStore(db)

	mock.ExpectDel("session:sess-123:ctx").SetVal(1)

	err := store.Delete(context.Background(), "sess-123")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionStore_KeyFormat(t *testing.T) {
	db, _ := redismock.NewClientMock()
	store := NewSessionStore(db)

	assert.Equal(t, "session:abc-def:ctx", store.key("abc-def"))
}
