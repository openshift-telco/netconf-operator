package v1

// DependsOn allows to specify a dependency for the operation to execute.
//If such dependency is not met, and/or if the underlying dependency isn't
//reporting success, the operation will fail.
type DependsOn struct {
	// Any of the Kind supported by netconf.adetalhouet.io/v1 Group
	Kind string `json:"kind,omitempty"`
	// The name of the object, which will be checked for within the same namespace
	Name string `json:"name,omitempty"`
}

func (d *DependsOn) IsNil() bool {
	return len(d.Kind) == 0 || len(d.Name) == 0
}
