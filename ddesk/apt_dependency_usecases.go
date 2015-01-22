package main

type AptDependencyManager interface {
	Store(*AptRepositoryAccess) error
	Remove(AptRepositoryId) error
	List() map[AptRepositoryId]*AptRepositoryAccess
}
