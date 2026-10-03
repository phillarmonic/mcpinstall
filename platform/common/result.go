package common

// The adapter contract returns paths and warnings as slices because a host may
// touch more than one file. These helpers keep the single-file case from
// littering every adapter with nil checks.

// Paths returns path as a one-element slice when the change happened, and nil
// otherwise.
func Paths(path string, changed bool) []string {
	if !changed {
		return nil
	}
	return []string{path}
}

// Warnings returns warning as a one-element slice when it is not empty, and nil
// otherwise.
func Warnings(warning string) []string {
	if warning == "" {
		return nil
	}
	return []string{warning}
}
