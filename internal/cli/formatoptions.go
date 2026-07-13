package cli

// constAppender is a pflag.Value that appends a fixed constant string onto
// a shared list whenever its flag is given, with no argument (like
// action='append_const' in argparse). Used so --sorted/--unsorted/
// --no-sorted/--no-unsorted interleave in command-line order with
// --format-options onto the same underlying list, mirroring httpie's own
// append_const-onto-shared-dest trick in cli/definition.py.
type constAppender struct {
	target *[]string
	value  string
}

func (a *constAppender) String() string   { return "" }
func (a *constAppender) Type() string     { return "bool" }
func (a *constAppender) IsBoolFlag() bool { return true }
func (a *constAppender) Set(string) error {
	*a.target = append(*a.target, a.value)
	return nil
}

// rawAppender is a pflag.Value that appends its raw string argument onto
// the same shared list as constAppender, for --format-options (which may
// be repeated, and each occurrence may itself be a comma-separated list).
type rawAppender struct {
	target *[]string
}

func (a *rawAppender) String() string { return "" }
func (a *rawAppender) Type() string   { return "string" }
func (a *rawAppender) Set(s string) error {
	*a.target = append(*a.target, s)
	return nil
}
