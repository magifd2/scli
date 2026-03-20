package keychain

import "fmt"

// MockStore is an in-memory Store implementation for use in tests.
type MockStore struct {
	data map[string]string
}

// NewMockStore returns an empty MockStore.
func NewMockStore() *MockStore {
	return &MockStore{data: make(map[string]string)}
}

// Get returns the token for the given workspace, or an error if not found.
func (m *MockStore) Get(workspace string) (string, error) {
	token, ok := m.data[workspace]
	if !ok {
		return "", fmt.Errorf("keychain: %q not found", workspace)
	}
	return token, nil
}

// Set stores the token for the given workspace.
func (m *MockStore) Set(workspace, token string) error {
	m.data[workspace] = token
	return nil
}

// Delete removes the token for the given workspace.
func (m *MockStore) Delete(workspace string) error {
	delete(m.data, workspace)
	return nil
}
