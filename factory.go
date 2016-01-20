package pool

import "errors"

// PooledObjectFactory is factory interface for ObjectPool
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

// DefaultPooledObjectFactory is a default PooledObjectFactory impl, support init by func
type DefaultPooledObjectFactory struct {
	make      func() (*PooledObject, error)
	destroy   func(object *PooledObject) error
	validate  func(object *PooledObject) bool
	activate  func(object *PooledObject) error
	passivate func(object *PooledObject) error
}

// NewPooledObjectFactorySimple return a DefaultPooledObjectFactory, only custom MakeObject func
func NewPooledObjectFactorySimple(
	create func() (interface{}, error)) PooledObjectFactory {
	return NewPooledObjectFactory(create, nil, nil, nil, nil)
}

// NewPooledObjectFactory return a DefaultPooledObjectFactory, init with gaven func.
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

// MakeObject see PooledObjectFactory.MakeObject
func (f *DefaultPooledObjectFactory) MakeObject() (*PooledObject, error) {
	return f.make()
}

// DestroyObject see PooledObjectFactory.DestroyObject
func (f *DefaultPooledObjectFactory) DestroyObject(object *PooledObject) error {
	if f.destroy != nil {
		return f.destroy(object)
	}
	return nil
}

// ValidateObject see PooledObjectFactory.ValidateObject
func (f *DefaultPooledObjectFactory) ValidateObject(object *PooledObject) bool {
	if f.validate != nil {
		return f.validate(object)
	}
	return true
}

// ActivateObject see PooledObjectFactory.ActivateObject
func (f *DefaultPooledObjectFactory) ActivateObject(object *PooledObject) error {
	if f.activate != nil {
		return f.activate(object)
	}
	return nil
}

// PassivateObject see PooledObjectFactory.PassivateObject
func (f *DefaultPooledObjectFactory) PassivateObject(object *PooledObject) error {
	if f.passivate != nil {
		return f.passivate(object)
	}
	return nil
}
