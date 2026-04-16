package redis

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheStore_Get_Success(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewCacheStore(db)

	mock.ExpectGet("key1").SetVal(`{"name":"Alice"}`)

	var dest map[string]string
	err := store.Get(context.Background(), "key1", &dest)

	require.NoError(t, err)
	assert.Equal(t, "Alice", dest["name"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCacheStore_Get_Miss(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewCacheStore(db)

	mock.ExpectGet("missing").RedisNil()

	var dest map[string]string
	err := store.Get(context.Background(), "missing", &dest)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCacheStore_Set_Success(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewCacheStore(db)

	// redis client sends []byte, not string
	mock.ExpectSet("key1", []byte(`{"v":1}`), 5*time.Minute).SetVal("OK")

	err := store.Set(context.Background(), "key1", map[string]int{"v": 1}, 5*time.Minute)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCacheStore_Delete(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewCacheStore(db)

	mock.ExpectDel("key1").SetVal(1)

	err := store.Delete(context.Background(), "key1")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCacheStore_Exists(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewCacheStore(db)

	mock.ExpectExists("key1").SetVal(1)

	exists, err := store.Exists(context.Background(), "key1")

	require.NoError(t, err)
	assert.True(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCacheStore_Exists_NotFound(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewCacheStore(db)

	mock.ExpectExists("key1").SetVal(0)

	exists, err := store.Exists(context.Background(), "key1")

	require.NoError(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}
