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
