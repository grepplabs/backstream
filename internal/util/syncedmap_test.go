package util

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyncedMap(t *testing.T) {
	sm := NewSyncedMap[string, int]()

	sm.Set("testKey", 1)
	val, ok := sm.Get("testKey")

	require.True(t, ok)
	require.Equal(t, 1, val)

	_, ok = sm.Get("someRandomKey")
	require.False(t, ok)

	sm.Delete("testKey")
	_, ok = sm.Get("testKey")
	require.False(t, ok)
}

func TestSyncedMap_Concurrency(t *testing.T) {
	sm := NewSyncedMap[int, int]()

	const count = 1000

	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sm.Set(i, i)
		}(i)
	}
	wg.Wait()

	for i := 0; i < count; i++ {
		v, ok := sm.Get(i)
		require.True(t, ok)
		require.Equal(t, i, v, "Incorrect value for key")
	}

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sm.Delete(i)
		}(i)
	}
	wg.Wait()
	require.Equal(t, sm.Size(), 0)
}
