package cmd

import (
	"github.com/nlink-jp/scli/internal/config"
	"github.com/nlink-jp/scli/internal/keychain"
	"github.com/nlink-jp/scli/internal/slack"
)

// Service factories — overridable in tests.
var (
	newKeychainStore func() keychain.Store           = func() keychain.Store { return &keychain.OSStore{} }
	newConfigManager func() (*config.Manager, error) = config.DefaultManager
	// newSlackClientOverride is non-nil only during tests.
	newSlackClientOverride func() (*slack.Client, error)
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

// SetSlackClientForTest replaces the Slack client factory for tests.
// Pass nil to clear the override. Call ResetServices in a defer.
func SetSlackClientForTest(fn func() (*slack.Client, error)) {
	newSlackClientOverride = fn
}

// ResetServices restores the service factories to their production defaults.
func ResetServices() {
	newKeychainStore = defaultKeychainStore
	newConfigManager = defaultConfigManager
	newSlackClientOverride = nil
}
