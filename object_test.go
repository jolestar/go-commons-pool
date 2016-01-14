package pool

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPooledObject(t *testing.T) {
	object := &TestObject{Num: 1}
	pooledObject := NewPooledObject(object)
	pooledObject.MarkReturning()
	assert.Equal(t, RETURNING, pooledObject.GetState())

	pooledObject.MarkAbandoned()
	assert.Equal(t, ABANDONED, pooledObject.GetState())
}
