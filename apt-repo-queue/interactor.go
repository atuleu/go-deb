package main

type Interactor struct {
	keyManager PgpKeyManager
	repo       AptRepo
}

func NewInteractor(opt *Options) (*Interactor, error) {
	config, err := LoadConfig(opt.Base)
	if err != nil {
		return nil, err
	}
	res := &Interactor{}
	res.keyManager, err = NewGpgKeyManager(config)
	if err != nil {
		return nil, err
	}
	res.repo, err = NewRepreproRepository(config)
	if err != nil {
		return nil, err
	}

	return res, nil
}
