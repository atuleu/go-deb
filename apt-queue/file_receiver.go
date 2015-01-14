package main

type Packagereceiver interface {
	Next() (string, error)
	Release(string)
}
