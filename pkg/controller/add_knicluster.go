package controller

import (
	"github.com/mhrivnak/kni-operator/pkg/controller/knicluster"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, knicluster.Add)
}
