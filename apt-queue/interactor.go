package main

type Interactor struct {
	keyManager PgpKeyManager
	repo       AptRepo
}

func NewInteractor(opt *Options) (*Interactor, error) {
	res := &Interactor{}
	var err error
	res.keyManager, err = NewHomeGpgKeyManager()
	if err != nil {
		return nil, err
	}

	return res, nil
}
