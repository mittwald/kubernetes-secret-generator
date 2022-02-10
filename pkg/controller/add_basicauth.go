package controller

import (
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/crd/basicauth"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, managerFunc{true, basicauth.Add})
}
