//go:build !noextendedalias

package poml

// extendedAliasesEnabled controls whether <extended-op>/<extended-figure> are accepted as aliases.
func extendedAliasesEnabled() bool { return true }
