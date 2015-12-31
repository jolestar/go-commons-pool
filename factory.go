package pool

import "errors"

type PooledObjectFactory interface {
	MakeObject() (*PooledObject, error)

	DestroyObject(object *PooledObject) error

	ValidateObject(object *PooledObject) bool

	ActivateObject(object *PooledObject) error

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
