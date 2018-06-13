package pool

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPoolConfig(t *testing.T) {
	config := NewDefaultPoolConfig()
	if debugTest {
		fmt.Println(config)
	}
	assert.NotNil(t, config)
}

func TestGetEvictionPolicy(t *testing.T) {
	policy := GetEvictionPolicy(DefaultEvictionPolicyName)
	assert.NotNil(t, policy)
}

func TestDefaultConfig(t *testing.T) {
	config := NewDefaultPoolConfig()
	assert.Equal(t, DefaultBlockWhenExhausted, config.BlockWhenExhausted)
	assert.Equal(t, DefaultEvictionPolicyName, config.EvictionPolicyName)
	assert.Equal(t, DefaultLifo, config.Lifo)
	assert.Equal(t, DefaultMaxIdle, config.MaxIdle)
	assert.Equal(t, DefaultMaxTotal, config.MaxTotal)
	assert.Equal(t, DefaultMinEvictableIdleTimeMillis, config.MinEvictableIdleTimeMillis)
	assert.Equal(t, DefaultMinIdle, config.MinIdle)
	assert.Equal(t, DefaultNumTestsPerEvictionRun, config.NumTestsPerEvictionRun)
	assert.Equal(t, DefaultSoftMinEvictableIdleTimeMillis, config.SoftMinEvictableIdleTimeMillis)
	assert.Equal(t, DefaultTestOnBorrow, config.TestOnBorrow)
	assert.Equal(t, DefaultTestOnCreate, config.TestOnCreate)
	assert.Equal(t, DefaultTestOnReturn, config.TestOnReturn)
	assert.Equal(t, DefaultTestWhileIdle, config.TestWhileIdle)
	assert.Equal(t, DefaultTimeBetweenEvictionRunsMillis, config.TimeBetweenEvictionRunsMillis)
}
