package main

import (
	"testing"

	. "gopkg.in/check.v1"
)

type UseCaseSuite struct {
	x Interactor
}

var _ = Suite(&UseCaseSuite{})

func Test(t *testing.T) { TestingT(t) }
