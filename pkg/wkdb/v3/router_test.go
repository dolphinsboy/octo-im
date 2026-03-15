package v3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewStaticBucketRouterRejectsZero(t *testing.T) {
	_, err := NewStaticBucketRouter(0)
	require.Error(t, err)
}

func TestStaticBucketRouterRoutesDeterministically(t *testing.T) {
	router, err := NewStaticBucketRouter(16)
	require.NoError(t, err)
	require.Equal(t, uint32(16), router.BucketCount())
	require.Equal(t, uint32(5), router.BucketForSlot(5))
	require.Equal(t, uint32(5), router.BucketForSlot(21))
	require.Equal(t, uint32(15), router.BucketForSlot(31))
}
