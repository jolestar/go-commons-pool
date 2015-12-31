package pool

import (
	"errors"
	"fmt"
	"sync"
)

const (
	DEFAULT_MAX_TOTAL = 8
	DEFAULT_MAX_IDLE  = 8
	DEFAULT_MIN_IDLE  = 0
	DEFAULT_LIFO      = true
	//DEFAULT_FAIRNESS = false
	DEFAULT_MAX_WAIT_MILLIS                     = int64(-1)
	DEFAULT_MIN_EVICTABLE_IDLE_TIME_MILLIS      = int64(1000 * 60 * 30)
	DEFAULT_SOFT_MIN_EVICTABLE_IDLE_TIME_MILLIS = int64(-1)
	DEFAULT_NUM_TESTS_PER_EVICTION_RUN          = 3
	DEFAULT_TEST_ON_CREATE                      = false
	DEFAULT_TEST_ON_BORROW                      = false
	DEFAULT_TEST_ON_RETURN                      = false
	DEFAULT_TEST_WHILE_IDLE                     = false
	DEFAULT_TIME_BETWEEN_EVICTION_RUNS_MILLIS   = int64(-1)
	DEFAULT_BLOCK_WHEN_EXHAUSTED                = true
	DEFAULT_EVICTION_POLICY_NAME                = "github.com/jolestar/go-commons-pool/DefaultEvictionPolicy"
)

type ObjectPoolConfig struct {
	// whether the pool has LIFO (last in, first out)
	Lifo     bool
	MaxTotal int
	MaxIdle  int
	MinIdle  int
	//TODO support fairness config
	//Fairness                       bool
	MaxWaitMillis                  int64
	MinEvictableIdleTimeMillis     int64
	SoftMinEvictableIdleTimeMillis int64

	NumTestsPerEvictionRun int

	EvictionPolicyName string

	TestOnCreate bool

	TestOnBorrow bool

	TestOnReturn bool

	TestWhileIdle bool

	TimeBetweenEvictionRunsMillis int64

	BlockWhenExhausted bool
}

func NewDefaultPoolConfig() *ObjectPoolConfig {
	return &ObjectPoolConfig{
		Lifo:                           DEFAULT_LIFO,
		MaxTotal:                       DEFAULT_MAX_TOTAL,
		MaxIdle:                        DEFAULT_MAX_IDLE,
		MinIdle:                        DEFAULT_MIN_IDLE,
		MaxWaitMillis:                  DEFAULT_MAX_WAIT_MILLIS,
		MinEvictableIdleTimeMillis:     DEFAULT_MIN_EVICTABLE_IDLE_TIME_MILLIS,
		SoftMinEvictableIdleTimeMillis: DEFAULT_SOFT_MIN_EVICTABLE_IDLE_TIME_MILLIS,

		NumTestsPerEvictionRun: DEFAULT_NUM_TESTS_PER_EVICTION_RUN,

		EvictionPolicyName: DEFAULT_EVICTION_POLICY_NAME,

		TestOnCreate: DEFAULT_TEST_ON_CREATE,

		TestOnBorrow: DEFAULT_TEST_ON_BORROW,

		TestOnReturn: DEFAULT_TEST_ON_RETURN,

		TestWhileIdle: DEFAULT_TEST_WHILE_IDLE,

		TimeBetweenEvictionRunsMillis: DEFAULT_TIME_BETWEEN_EVICTION_RUNS_MILLIS,
		BlockWhenExhausted:            DEFAULT_BLOCK_WHEN_EXHAUSTED}
}

type AbandonedConfig struct {
	RemoveAbandonedOnBorrow      bool
	RemoveAbandonedOnMaintenance bool
	// Timeout in seconds before an abandoned object can be removed.
	RemoveAbandonedTimeout int
}

type EvictionConfig struct {
	IdleEvictTime     int64
	IdleSoftEvictTime int64
	MinIdle           int
}

type EvictionPolicy interface {
	Evict(config *EvictionConfig, underTest *PooledObject, idleCount int) bool
}

type DefaultEvictionPolicy struct {
}

func (this *DefaultEvictionPolicy) Evict(config *EvictionConfig, underTest *PooledObject, idleCount int) bool {
	if (config.IdleSoftEvictTime < underTest.GetIdleTimeMillis() &&
		config.MinIdle < idleCount) ||
		config.IdleEvictTime < underTest.GetIdleTimeMillis() {
		return true
	}
	return false
}

var (
	policiesMutex sync.Mutex
	policies      = make(map[string]EvictionPolicy)
)

func RegistryEvictionPolicy(name string, policy EvictionPolicy) {
	if name == "" || policy == nil {
		panic(errors.New("invalid argument"))
	}
	fmt.Println("RegistryEvictionPolicy", name)
	policiesMutex.Lock()
	policies[name] = policy
	policiesMutex.Unlock()
}

func GetEvictionPolicy(name string) EvictionPolicy {
	if name == "" {
		return nil
	}
	policiesMutex.Lock()
	defer policiesMutex.Unlock()
	return policies[name]

}

func init() {
	RegistryEvictionPolicy(DEFAULT_EVICTION_POLICY_NAME, new(DefaultEvictionPolicy))
}
