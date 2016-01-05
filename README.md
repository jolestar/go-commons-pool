Go Commons Pool
=====

[![Build Status](https://travis-ci.org/jolestar/go-commons-pool.svg?branch=master)](https://travis-ci.org/jolestar/go-commons-pool)
[![Circle CI](https://circleci.com/gh/jolestar/go-commons-pool.svg?style=svg)](https://circleci.com/gh/jolestar/go-commons-pool)
[![Coverage Status](https://coveralls.io/repos/jolestar/go-commons-pool/badge.svg?branch=master&service=github&_day=201606)](https://coveralls.io/github/jolestar/go-commons-pool?branch=master)
[![GoDoc](http://godoc.org/github.com/jolestar/go-commons-pool?status.svg)](http://godoc.org/github.com/jolestar/go-commons-pool)

The Go Commons Pool is a [Go](http://golang.org/) generic object pool, direct translate from [Apache Commons Pool](https://commons.apache.org/proper/commons-pool/).


Features
-------
* Support custom [PooledObjectFactory](https://godoc.org/github.com/jolestar/go-commons-pool#PooledObjectFactory)
* Rich pool configuration option, can precise control pooled object lifecycle. see [ObjectPoolConfig](https://godoc.org/github.com/jolestar/go-commons-pool#ObjectPoolConfig).

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
