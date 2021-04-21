package rest

import (
	"github.com/go-kit/kit/log"
	"github.com/hhio618/go-golem/pkg/logging"
	"github.com/pkg/errors"
)

const ComponentName = "rest"

// Package level logger.
var logger log.Logger

func init() {

	filterLog, err := logging.ApplyFilter(ComponentName, logging.NewLogger())
	if err != nil {
		panic(errors.Wrap(err, "apply filter logger"))
	}
	logger = log.With(filterLog, "component", ComponentName)
}
