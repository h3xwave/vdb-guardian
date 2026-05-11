package migration

// Default collection, table, and field names shared across seeders, migration
// adapters, and their unit tests so repeated string literals don't trigger
// golangci-lint goconst warnings.
const (
	DefaultSeedCollection  = "items"
	DefaultSeedIDField     = "id"
	DefaultSeedVectorField = "embedding"
)
