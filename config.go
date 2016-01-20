package pool

import (
	"errors"
	"fmt"
	"math"
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
	DEFAULT_SOFT_MIN_EVICTABLE_IDLE_TIME_MILLIS = int64(math.MaxInt64)
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
	/**
	 * Whether the pool has LIFO (last in, first out) behaviour with
	 * respect to idle objects - always returning the most recently used object
	 * from the pool, or as a FIFO (first in, first out) queue, where the pool
	 * always returns the oldest object in the idle object pool.
	 */
	Lifo bool

	/**
	 * The cap on the number of objects that can be allocated by the pool
	 * (checked out to clients, or idle awaiting checkout) at a given time. Use
	 * a negative value for no limit.
	 */
	MaxTotal int

	/**
	 * The cap on the number of "idle" instances in the pool. Use a
	 * negative value to indicate an unlimited number of idle instances.
	 * If MaxIdle
	 * is set too low on heavily loaded systems it is possible you will see
	 * objects being destroyed and almost immediately new objects being created.
	 * This is a result of the active goroutines momentarily returning objects
	 * faster than they are requesting them them, causing the number of idle
	 * objects to rise above maxIdle. The best value for maxIdle for heavily
	 * loaded system will vary but the default is a good starting point.
	 */
	MaxIdle int

	/**
	 * The minimum number of idle objects to maintain in
	 * the pool. This setting only has an effect if it is positive and
	 * TimeBetweenEvictionRunsMillis is greater than zero. If this
	 * is the case, an attempt is made to ensure that the pool has the required
	 * minimum number of instances during idle object eviction runs.
	 *
	 * If the configured value of MinIdle is greater than the configured value
	 * for MaxIdle then the value of MaxIdle will be used instead.
	 *
	 */
	MinIdle int

	/**
	* Whether objects created for the pool will be validated before
	* being returned from the ObjectPool.BorrowObject() method. Validation is
	* performed by the ValidateObject() method of the factory
	* associated with the pool. If the object fails to validate, then
	* ObjectPool.BorrowObject() will fail.
	 */
	TestOnCreate bool

	/**
	 * Whether objects borrowed from the pool will be validated before
	 * being returned from the ObjectPool.BorrowObject() method. Validation is
	 * performed by the ValidateObject() method of the factory
	 * associated with the pool. If the object fails to validate, it will be
	 * removed from the pool and destroyed, and a new attempt will be made to
	 * borrow an object from the pool.
	 */
	TestOnBorrow bool

	/**
	 * Whether objects borrowed from the pool will be validated when
	 * they are returned to the pool via the ObjectPool.ReturnObject() method.
	 * Validation is performed by the ValidateObject() method of
	 * the factory associated with the pool. Returning objects that fail validation
	 * are destroyed rather then being returned the pool.
	 */
	TestOnReturn bool

	/**
	* Whether objects sitting idle in the pool will be validated by the
	* idle object evictor (if any - see
	*  TimeBetweenEvictionRunsMillis ). Validation is performed
	* by the ValidateObject() method of the factory associated
	* with the pool. If the object fails to validate, it will be removed from
	* the pool and destroyed.  Note that setting this property has no effect
	* unless the idle object evictor is enabled by setting
	* TimeBetweenEvictionRunsMillis  to a positive value.
	 */
	TestWhileIdle bool

	/**
	* Whether to block when the ObjectPool.BorrowObject() method is
	* invoked when the pool is exhausted (the maximum number of "active"
	* objects has been reached).
	 */
	BlockWhenExhausted bool

	//TODO support fairness config
	//Fairness                       bool

	/**
	 * The maximum amount of time (in milliseconds) the
	 * ObjectPool.BorrowObject() method should block before return
	 * a error when the pool is exhausted and
	 *  BlockWhenExhausted is true. When less than 0, the
	 * ObjectPool.BorrowObject() method may block indefinitely.
	 *
	 */
	MaxWaitMillis int64

	/**
	 * The minimum amount of time an object may sit idle in the pool
	 * before it is eligible for eviction by the idle object evictor (if any -
	 * see TimeBetweenEvictionRunsMillis . When non-positive,
	 * no objects will be evicted from the pool due to idle time alone.
	 */
	MinEvictableIdleTimeMillis int64

	/**
	 * The minimum amount of time an object may sit idle in the pool
	 * before it is eligible for eviction by the idle object evictor (if any -
	 * see TimeBetweenEvictionRunsMillis ),
	 * with the extra condition that at least MinIdle object
	 * instances remain in the pool. This setting is overridden by
	 *  MinEvictableIdleTimeMillis (that is, if
	 *  MinEvictableIdleTimeMillis is positive, then
	 *  SoftMinEvictableIdleTimeMillis is ignored).
	 */
	SoftMinEvictableIdleTimeMillis int64

	/**
	 * The maximum number of objects to examine during each run (if any)
	 * of the idle object evictor goroutine. When positive, the number of tests
	 * performed for a run will be the minimum of the configured value and the
	 * number of idle instances in the pool. When negative, the number of tests
	 * performed will be math.Ceil(ObjectPool.GetNumIdle()/math.
	 * Abs(PoolConfig.NumTestsPerEvictionRun)) which means that when the
	 * value is -n roughly one nth of the idle objects will be
	 * tested per run.
	 */
	NumTestsPerEvictionRun int

	/**
	 * The name of the EvictionPolicy implementation that is
	 * used by this pool. Please register policy by RegistryEvictionPolicy(name, policy)
	 */
	EvictionPolicyName string

	/**
	* The number of milliseconds to sleep between runs of the idle
	* object evictor goroutine. When non-positive, no idle object evictor goroutine
	* will be run.
	* if this value changed after ObjectPool created, should call ObjectPool.StartEvictor to take effect.
	 */
	TimeBetweenEvictionRunsMillis int64
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
		NumTestsPerEvictionRun:         DEFAULT_NUM_TESTS_PER_EVICTION_RUN,
		EvictionPolicyName:             DEFAULT_EVICTION_POLICY_NAME,
		TestOnCreate:                   DEFAULT_TEST_ON_CREATE,
		TestOnBorrow:                   DEFAULT_TEST_ON_BORROW,
		TestOnReturn:                   DEFAULT_TEST_ON_RETURN,
		TestWhileIdle:                  DEFAULT_TEST_WHILE_IDLE,
		TimeBetweenEvictionRunsMillis:  DEFAULT_TIME_BETWEEN_EVICTION_RUNS_MILLIS,
		BlockWhenExhausted:             DEFAULT_BLOCK_WHEN_EXHAUSTED}
}

type AbandonedConfig struct {
	RemoveAbandonedOnBorrow      bool
	RemoveAbandonedOnMaintenance bool
	// Timeout in seconds before an abandoned object can be removed.
	RemoveAbandonedTimeout int
}

func NewDefaultAbandonedConfig() *AbandonedConfig {
	return &AbandonedConfig{RemoveAbandonedOnBorrow: false, RemoveAbandonedOnMaintenance: false, RemoveAbandonedTimeout: 300}
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

func (p *DefaultEvictionPolicy) Evict(config *EvictionConfig, underTest *PooledObject, idleCount int) bool {
	idleTime := underTest.GetIdleTimeMillis()

	if (config.IdleSoftEvictTime < idleTime &&
		config.MinIdle < idleCount) ||
		config.IdleEvictTime < idleTime {
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
	policiesMutex.Lock()
	defer policiesMutex.Unlock()
	return policies[name]

}

func init() {
	RegistryEvictionPolicy(DEFAULT_EVICTION_POLICY_NAME, new(DefaultEvictionPolicy))
}
