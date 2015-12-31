package pool

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestDefaultPooledObjectFactory(t *testing.T) {
	factory := NewPooledObjectFactorySimple(
		func() (interface{}, error) {
			return &TestObject{Num: rand.Int()}, nil
		})

	assert.NotNil(t, factory)
	o, _ := factory.MakeObject()
	fmt.Println("object:", o.Object)
	assert.NotNil(t, o)
	assert.Nil(t, factory.ActivateObject(o))
	assert.Nil(t, factory.PassivateObject(o))
	assert.True(t, factory.ValidateObject(o))
	assert.Nil(t, factory.DestroyObject(o))
}
