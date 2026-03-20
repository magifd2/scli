// Package keychain provides an abstraction over OS-native secret storage.
package keychain

import "github.com/zalando/go-keyring"

const serviceName = "scli"

// Store defines the interface for OS secret storage.
// Implementations include OSStore (production) and MockStore (tests).
type Store interface {
	Get(workspace string) (string, error)
	Set(workspace, token string) error
	Delete(workspace string) error
}

// OSStore uses the native OS keychain (macOS Keychain, Linux libsecret, Windows Credential Manager).
type OSStore struct{}

// Get retrieves the token for the given workspace from the OS keychain.
func (s *OSStore) Get(workspace string) (string, error) {
	return keyring.Get(serviceName, workspace)
}

// Set stores the token for the given workspace in the OS keychain.
func (s *OSStore) Set(workspace, token string) error {
	return keyring.Set(serviceName, workspace, token)
}

// Delete removes the token for the given workspace from the OS keychain.
func (s *OSStore) Delete(workspace string) error {
	return keyring.Delete(serviceName, workspace)
}
