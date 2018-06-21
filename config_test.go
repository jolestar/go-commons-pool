package pool

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPoolConfig(t *testing.T) {
	t.Parallel()

	config := NewDefaultPoolConfig()
	if debugTest {
		fmt.Println(config)
	}
	assert.NotNil(t, config)
}

func TestGetEvictionPolicy(t *testing.T) {
	t.Parallel()

	policy := GetEvictionPolicy(DefaultEvictionPolicyName)
	assert.NotNil(t, policy)
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	config := NewDefaultPoolConfig()
	assert.Equal(t, DefaultBlockWhenExhausted, config.BlockWhenExhausted)
	assert.Equal(t, DefaultEvictionPolicyName, config.EvictionPolicyName)
	assert.Equal(t, DefaultLIFO, config.LIFO)
	assert.Equal(t, DefaultMaxIdle, config.MaxIdle)
	assert.Equal(t, DefaultMaxTotal, config.MaxTotal)
	assert.Equal(t, DefaultMinEvictableIdleTime, config.MinEvictableIdleTime)
	assert.Equal(t, DefaultMinIdle, config.MinIdle)
	assert.Equal(t, DefaultNumTestsPerEvictionRun, config.NumTestsPerEvictionRun)
	assert.Equal(t, DefaultSoftMinEvictableIdleTime, config.SoftMinEvictableIdleTime)
	assert.Equal(t, DefaultTestOnBorrow, config.TestOnBorrow)
	assert.Equal(t, DefaultTestOnCreate, config.TestOnCreate)
	assert.Equal(t, DefaultTestOnReturn, config.TestOnReturn)
	assert.Equal(t, DefaultTestWhileIdle, config.TestWhileIdle)
	assert.Equal(t, DefaultTimeBetweenEvictionRuns, config.TimeBetweenEvictionRuns)
}
