package config

import (
	"fmt"

	keyring "github.com/zalando/go-keyring"
)

const keychainService = "vpn-kongtrol"

// GetCredential retrieves a credential from the OS keychain.
// service is scoped to the VPN profile name; key is "username" or "password".
func GetCredential(profile, key string) (string, error) {
	item := fmt.Sprintf("%s.%s", profile, key)
	val, err := keyring.Get(keychainService, item)
	if err != nil {
		return "", fmt.Errorf("keychain: get %s/%s: %w", profile, key, err)
	}
	return val, nil
}

// SetCredential stores a credential in the OS keychain.
func SetCredential(profile, key, value string) error {
	item := fmt.Sprintf("%s.%s", profile, key)
	if err := keyring.Set(keychainService, item, value); err != nil {
		return fmt.Errorf("keychain: set %s/%s: %w", profile, key, err)
	}
	return nil
}

// DeleteCredential removes a credential from the OS keychain.
func DeleteCredential(profile, key string) error {
	item := fmt.Sprintf("%s.%s", profile, key)
	if err := keyring.Delete(keychainService, item); err != nil {
		return fmt.Errorf("keychain: delete %s/%s: %w", profile, key, err)
	}
	return nil
}
