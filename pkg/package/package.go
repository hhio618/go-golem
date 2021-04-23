package pkg

import "github.com/hhio618/go-golem/pkg/props"

type PackageError struct {
	e error
}

func (e PackageError) Error() string {
	return e.e.Error()
}

type Package interface {
	ResolveUrl() (string, error)
	DecorateDemand(deman *props.DemandBuilder) error
}
