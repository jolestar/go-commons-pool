Go Commons Pool
=====

[![Build Status](https://travis-ci.org/jolestar/go-commons-pool.svg?branch=master)](https://travis-ci.org/jolestar/go-commons-pool)

The Go Commons Pool is a [Go](http://golang.org/) generic object pool, direct translate from [Apache Commons Pool](https://commons.apache.org/proper/commons-pool/).


Features
-------
* Object factory API
* Rich pool configuration option 

Usage
-------

    //use create func
    pool := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
    		func() (interface{}, error) {
    			return &MyPoolObject{}, nil
    		}))
    obj, _ := pool.BorrowObject()
    pool.ReturnObject(obj)
    	
    //use custom Object factory
    
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
    
    pool := NewObjectPoolWithDefaultConfig(new(MyObjectFactory))
    obj, _ := pool.BorrowObject()
    pool.ReturnObject(obj)

TODO
-------
* Add more unit test
* Add benchmark test
* Support go document
* Refactor Java style to Go style

License
-------

Go Commons Pool is available under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.html).
