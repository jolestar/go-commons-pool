Go Commons Pool
=====

[![Build Status](https://travis-ci.org/jolestar/go-commons-pool.svg?branch=master)](https://travis-ci.org/jolestar/go-commons-pool)
[![CodeCov](https://codecov.io/gh/jolestar/go-commons-pool/branch/master/graph/badge.svg)](https://codecov.io/gh/jolestar/go-commons-pool)
[![GoDoc](http://godoc.org/github.com/jolestar/go-commons-pool?status.svg)](http://godoc.org/github.com/jolestar/go-commons-pool)

The Go Commons Pool is a generic object pool for [Golang](http://golang.org/), direct rewrite from [Apache Commons Pool](https://commons.apache.org/proper/commons-pool/).


Features
-------
1. Support custom [PooledObjectFactory](https://godoc.org/github.com/jolestar/go-commons-pool#PooledObjectFactory).
1. Rich pool configuration option, can precise control pooled object lifecycle. see [ObjectPoolConfig](https://godoc.org/github.com/jolestar/go-commons-pool#ObjectPoolConfig).
	* Pool LIFO (last in, first out) or FIFO (first in, first out) 
	* Pool cap config
	* Pool object validate config
	* Pool object borrow block and max waiting time config
	* Pool object eviction config
	* Pool object abandon config

Pool Configuration Option
-------

Configuration option table, more detail description see [ObjectPoolConfig](https://godoc.org/github.com/jolestar/go-commons-pool#ObjectPoolConfig)

| Option                        | Default        | Description  |
| ------------------------------|:--------------:| :------------|
| Lifo                          | true           |If pool is LIFO (last in, first out)|
| MaxTotal                      | 8              |The cap of pool|
| MaxIdle                       | 8              |Max "idle" instances in the pool |
| MinIdle                       | 0              |Min "idle" instances in the pool |
| TestOnCreate                  | false          |Validate when object is created|
| TestOnBorrow                  | false          |Validate when object is borrowed|
| TestOnReturn                  | false          |Validate when object is returned|
| TestWhileIdle                 | false          |Validate when object is idle, see TimeBetweenEvictionRuns |
| BlockWhenExhausted            | true           |Whether to block when the pool is exhausted  |
| MinEvictableIdleTime          | 30m            |Eviction configuration,see DefaultEvictionPolicy |
| SoftMinEvictableIdleTime      | math.MaxInt64  |Eviction configuration,see DefaultEvictionPolicy  |
| NumTestsPerEvictionRun        | 3              |The maximum number of objects to examine during each run evictor goroutine |
| TimeBetweenEvictionRuns       | 0              |The number of milliseconds to sleep between runs of the evictor goroutine, less than 1 mean not run |

Usage
-------
    import "github.com/jolestar/go-commons-pool"

    //use create func
    p := pool.NewObjectPoolWithDefaultConfig(pool.NewPooledObjectFactorySimple(
    		func() (interface{}, error) {
    			return &MyPoolObject{}, nil
    		}))
    obj, _ := p.BorrowObject()
    p.ReturnObject(obj)
    	
    //use custom Object factory
    
    type MyObjectFactory struct {
    }
    
    func (f *MyObjectFactory) MakeObject() (*pool.PooledObject, error) {
    	return pool.NewPooledObject(&MyPoolObject{}), nil
    }
    
    func (f *MyObjectFactory) DestroyObject(object *PooledObject) error {
    	//do destroy
    	return nil
    }
    
    func (f *MyObjectFactory) ValidateObject(object *pool.PooledObject) bool {
    	//do validate
    	return true
    }
    
    func (f *MyObjectFactory) ActivateObject(object *pool.PooledObject) error {
    	//do activate
    	return nil
    }
    
    func (f *MyObjectFactory) PassivateObject(object *pool.PooledObject) error {
    	//do passivate
    	return nil
    }
    
    p := pool.NewObjectPoolWithDefaultConfig(new(MyObjectFactory))
    p.Config.MaxTotal = 100
    obj, _ := p.BorrowObject()
    p.ReturnObject(obj)

more example please see pool_test.go and example_test.go

Note
-------
PooledObjectFactory.MakeObject must return a pointer, not value.
The following code will complain error. 

	p := pool.NewObjectPoolWithDefaultConfig(pool.NewPooledObjectFactorySimple(
		func() (interface{}, error) {
			return "hello", nil
		}))
	obj, _ := p.BorrowObject()
	p.ReturnObject(obj)

The right way is:

	p := pool.NewObjectPoolWithDefaultConfig(pool.NewPooledObjectFactorySimple(
		func() (interface{}, error) {
			var stringPointer = new(string)
			*stringPointer = "hello"
			return stringPointer, nil
		}))

more example please see example_test.go

Dependency
-------
* [testify](https://github.com/stretchr/testify) for test

PerformanceTest
-------
The results of running the pool_perf_test is almost equal to the java version [PerformanceTest](https://github.com/apache/commons-pool/blob/trunk/src/test/java/org/apache/commons/pool2/performance/PerformanceTest.java)
    
    go test --perf=true

For Apache commons pool user
-------
* Direct use pool.Config.xxx to change pool config
* Default config value is same as java version
* If TimeBetweenEvictionRuns changed after ObjectPool created, should call  **ObjectPool.StartEvictor** to take effect. Java version do this on set method.
* No KeyedObjectPool (TODO)
* No ProxiedObjectPool
* No pool stats (TODO)

FAQ
-------
[FAQ](https://github.com/jolestar/go-commons-pool/wiki/FAQ)

How to contribute
-------
* Choose one open issue you want to solve, if not create one and describe what you want to change.
* Fork the repository on GitHub.
* Write code to solve the issue.
* Create PR and link to the issue.
* Make sure test and coverage pass.
* Wait maintainers to merge.

中文文档
-------
* [Go Commons Pool发布以及Golang多线程编程问题总结](http://jolestar.com/go-commons-pool-and-go-concurrent/)
* [Go Commons Pool 1.0 发布](http://jolestar.com/go-commons-pool-v1-release/)


License
-------

Go Commons Pool is available under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.html).
