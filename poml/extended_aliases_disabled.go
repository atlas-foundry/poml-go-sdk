//go:build noextendedalias

package poml

// extendedAliasesEnabled controls whether <extended-op>/<extended-figure> aliases are accepted.
func extendedAliasesEnabled() bool { return false }
