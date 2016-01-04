package pool

import "testing"

type MyPoolObject struct {
}

func TestExample(t *testing.T) {
	pool := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
		func() (interface{}, error) {
			return &MyPoolObject{}, nil
		}))
	obj, _ := pool.BorrowObject()
	pool.ReturnObject(obj)
}

type MyObjectFactory struct {
}

func (this *MyObjectFactory) MakeObject() (*PooledObject, error) {
	return NewPooledObject(&MyPoolObject{}), nil
}

func (this *MyObjectFactory) DestroyObject(object *PooledObject) error {
	//do destroy
	return nil
}

func (this *MyObjectFactory) ValidateObject(object *PooledObject) bool {
	//do validate
	return true
}

func (this *MyObjectFactory) ActivateObject(object *PooledObject) error {
	//do activate
	return nil
}

func (this *MyObjectFactory) PassivateObject(object *PooledObject) error {
	//do passivate
	return nil
}

func TestCustomFactoryExample(t *testing.T) {
	pool := NewObjectPoolWithDefaultConfig(new(MyObjectFactory))
	obj, _ := pool.BorrowObject()
	pool.ReturnObject(obj)
}

//func TestStringExampleFail(t *testing.T) {
//	pool := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
//		func() (interface{}, error) {
//			return "hello", nil
//		}))
//	obj, _ := pool.BorrowObject()
//	pool.ReturnObject(obj)
//}

//func TestIntExampleFail(t *testing.T) {
//	pool := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
//		func() (interface{}, error) {
//			return 1, nil
//		}))
//	obj, _ := pool.BorrowObject()
//	pool.ReturnObject(obj)
//}
