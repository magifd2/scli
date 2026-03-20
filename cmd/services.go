package cmd

import (
	"github.com/magifd2/scli/internal/config"
	"github.com/magifd2/scli/internal/keychain"
)

// Service factories — overridable in tests.
var (
	newKeychainStore func() keychain.Store           = func() keychain.Store { return &keychain.OSStore{} }
	newConfigManager func() (*config.Manager, error) = config.DefaultManager
)

// defaultKeychainStore and defaultConfigManager hold the original production factories.
var (
	defaultKeychainStore = newKeychainStore
	defaultConfigManager = newConfigManager
)

// SetServicesForTest replaces the service factories with test doubles.
// Call ResetServices in a defer to restore them after the test.
func SetServicesForTest(ks keychain.Store, mgr *config.Manager) {
	newKeychainStore = func() keychain.Store { return ks }
	newConfigManager = func() (*config.Manager, error) { return mgr, nil }
}

// ResetServices restores the service factories to their production defaults.
func ResetServices() {
	newKeychainStore = defaultKeychainStore
	newConfigManager = defaultConfigManager
}
