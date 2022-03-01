package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type managerFunc struct {
	isCRD       bool
	managerFunc func(manager.Manager) error
}

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []managerFunc

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, enableCRDs bool) error {
	for _, mf := range AddToManagerFuncs {
		if enableCRDs || !mf.isCRD {
			if err := mf.managerFunc(m); err != nil {
				return err
			}
		}
	}
	return nil
}
