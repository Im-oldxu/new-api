package main

import (
	"github.com/Im-oldxu/new-api/examples/tickets"
	"github.com/Im-oldxu/new-api/router"
)

func init() {
	businessModules = append(businessModules, router.BusinessModule{Migrate: tickets.Migrate, Register: tickets.Register})
}
