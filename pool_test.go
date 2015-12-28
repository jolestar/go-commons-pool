package pool
import (
"github.com/stretchr/testify/suite"
"testing"
"github.com/stretchr/testify/assert"
)


type PoolTestSuite struct {
	suite.Suite
	pool *ObjectPool
}

func TestPoolTestSuite(t *testing.T) {
	suite.Run(t, new(PoolTestSuite))
}

func (suite *PoolTestSuite) SetupTest() {
	
}

type TestObject struct {
	num int
}

func makeEmptyPool(maxTotal int) *ObjectPool {
	pool := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
		func()(interface{},error) {
			return &TestObject{num:0},nil
		}))
	pool.PoolConfig.MaxTotal = maxTotal
	return pool
}

func getNthObject(num int) *TestObject {
	return &TestObject{num:0}
}

func (suite *PoolTestSuite)  TestBaseBorrow(){
	suite.pool = makeEmptyPool(3);
	o0,err := suite.pool.BorrowObject()

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), o0);

	assert.Equal(suite.T(),getNthObject(0), o0)
	o1,_ := suite.pool.BorrowObject()
	assert.Equal(suite.T(),getNthObject(1), o1)
	o2,_ := suite.pool.BorrowObject()
	assert.Equal(suite.T(),getNthObject(2), o2)
	suite.pool.Close()
}