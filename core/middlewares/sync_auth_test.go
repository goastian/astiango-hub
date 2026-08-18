package middlewares

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSyncNonceIsSingleUseAndExpires(t *testing.T) {
	node, nonce := "node-test", NewSyncNonce()
	require.NotEmpty(t, nonce)
	require.True(t, ConsumeSyncNonce(node, nonce))
	require.False(t, ConsumeSyncNonce(node, nonce))
	require.False(t, ConsumeSyncNonce(node, ""))

	syncNonces.Lock()
	syncNonces.values["expired:nonce"] = time.Now().Add(-time.Second)
	syncNonces.Unlock()
	require.True(t, ConsumeSyncNonce("expired", "nonce"))
}

func TestSyncRateLimitIsPerNode(t *testing.T) {
	for range syncNodeRateLimit {
		require.True(t, allowSyncNodeRequest("rate-node-a"))
	}
	require.False(t, allowSyncNodeRequest("rate-node-a"))
	require.True(t, allowSyncNodeRequest("rate-node-b"))
}
