package pool
import (
"github.com/stretchr/testify/assert"
"testing"
	"fmt"
)


func TestPoolConfig(t *testing.T) {
	config := NewDefaultPoolConfig()
	fmt.Println(config)
	assert.NotNil(t, config)
}

func TestGetEvictionPolicy(t *testing.T)  {
	policy := GetEvictionPolicy(DEFAULT_EVICTION_POLICY_NAME)
	assert.NotNil(t, policy)
}

func TestDefaultConfig(t *testing.T)  {
	config := NewDefaultPoolConfig()
	assert.Equal(t, DEFAULT_BLOCK_WHEN_EXHAUSTED, config.BlockWhenExhausted)
	assert.Equal(t, DEFAULT_EVICTION_POLICY_NAME, config.EvictionPolicyName)
	assert.Equal(t, DEFAULT_LIFO, config.Lifo)
	assert.Equal(t, DEFAULT_MAX_IDLE, config.MaxIdle)
	assert.Equal(t, DEFAULT_MAX_TOTAL, config.MaxTotal)
	assert.Equal(t, DEFAULT_MAX_WAIT_MILLIS, config.MaxWaitMillis)
	assert.Equal(t, DEFAULT_MIN_EVICTABLE_IDLE_TIME_MILLIS, config.MinEvictableIdleTimeMillis)
	assert.Equal(t, DEFAULT_MIN_IDLE, config.MinIdle)
	assert.Equal(t, DEFAULT_NUM_TESTS_PER_EVICTION_RUN, config.NumTestsPerEvictionRun)
	assert.Equal(t, DEFAULT_SOFT_MIN_EVICTABLE_IDLE_TIME_MILLIS, config.SoftMinEvictableIdleTimeMillis)
	assert.Equal(t, DEFAULT_TEST_ON_BORROW, config.TestOnBorrow)
	assert.Equal(t, DEFAULT_TEST_ON_CREATE, config.TestOnCreate)
	assert.Equal(t, DEFAULT_TEST_ON_RETURN, config.TestOnReturn)
	assert.Equal(t, DEFAULT_TEST_WHILE_IDLE, config.TestWhileIdle)
	assert.Equal(t, DEFAULT_TIME_BETWEEN_EVICTION_RUNS_MILLIS, config.TimeBetweenEvictionRunsMillis)
}