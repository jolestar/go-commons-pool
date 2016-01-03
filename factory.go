package pool

import "errors"

type PooledObjectFactory interface {

	/**
	 * Create an instance that can be served by the pool and wrap it in a
	 * PooledObject to be managed by the pool.
	 *
	 * return error if there is a problem creating a new instance,
	 *    this will be propagated to the code requesting an object.
	 */
	MakeObject() (*PooledObject, error)

	/**
	 * Destroys an instance no longer needed by the pool.
	 */
	DestroyObject(object *PooledObject) error

	/**
	 * Ensures that the instance is safe to be returned by the pool.
	 *
	 * return false if object is not valid and should
	 *         be dropped from the pool, true otherwise.
	 */
	ValidateObject(object *PooledObject) bool

	/**
	 * Reinitialize an instance to be returned by the pool.
	 *
	 * return error if there is a problem activating object,
	 *    this error may be swallowed by the pool.
	 */
	ActivateObject(object *PooledObject) error

	/**
	 * Uninitialize an instance to be returned to the idle object pool.
	 *
	 * return error if there is a problem passivating obj,
	 *    this exception may be swallowed by the pool.
	 */
	PassivateObject(object *PooledObject) error
}

type DefaultPooledObjectFactory struct {
	make      func() (*PooledObject, error)
	destroy   func(object *PooledObject) error
	validate  func(object *PooledObject) bool
	activate  func(object *PooledObject) error
	passivate func(object *PooledObject) error
}

func NewPooledObjectFactorySimple(
	create func() (interface{}, error)) PooledObjectFactory {
	return NewPooledObjectFactory(create, nil, nil, nil, nil)
}

func NewPooledObjectFactory(
	create func() (interface{}, error),
	destroy func(object *PooledObject) error,
	validate func(object *PooledObject) bool,
	activate func(object *PooledObject) error,
	passivate func(object *PooledObject) error) PooledObjectFactory {
	if create == nil {
		panic(errors.New("make function can not be nil"))
	}
	return &DefaultPooledObjectFactory{
		make: func() (*PooledObject, error) {
			o, err := create()
			if err != nil {
				return nil, err
			}
			return NewPooledObject(o), nil
		},
		destroy:   destroy,
		validate:  validate,
		activate:  activate,
		passivate: passivate}
}

func (this *DefaultPooledObjectFactory) MakeObject() (*PooledObject, error) {
	if this.make != nil {
		return this.make()
	} else {
		return nil, errors.New("object factory in illegal status")
	}
}

func (this *DefaultPooledObjectFactory) DestroyObject(object *PooledObject) error {
	if this.destroy != nil {
		return this.destroy(object)
	}
	return nil
}

func (this *DefaultPooledObjectFactory) ValidateObject(object *PooledObject) bool {
	if this.validate != nil {
		return this.validate(object)
	}
	return true
}

func (this *DefaultPooledObjectFactory) ActivateObject(object *PooledObject) error {
	if this.activate != nil {
		return this.activate(object)
	}
	return nil
}

func (this *DefaultPooledObjectFactory) PassivateObject(object *PooledObject) error {
	if this.passivate != nil {
		return this.passivate(object)
	}
	return nil
}
