package cli

// usageError is a command-line usage mistake (e.g. a missing required
// argument), rendered in httpie's "usage / error / for more information"
// format rather than cobra's terser default.
type usageError struct {
	usage    string
	progName string
	message  string
}

func (e *usageError) Error() string { return e.message }
