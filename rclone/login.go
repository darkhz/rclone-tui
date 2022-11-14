package rclone

import (
	"fmt"

	"golang.org/x/sync/semaphore"
)

var loginLock *semaphore.Weighted

// Login connects to the provided host with the username and password.
func Login(host, user, pass string) (string, error) {
	if loginLock == nil {
		loginLock = semaphore.NewWeighted(1)
	}

	if !loginLock.TryAcquire(1) {
		return "", fmt.Errorf("Attempting to log in")
	}
	defer loginLock.Release(1)

	CancelClientContext()

	err := SetupClient(host, user, pass)
	if err != nil {
		return "", err
	}

	_, err = GetVersion(true)
	if err != nil {
		return "", err
	}

	client, err := GetCurrentClient()
	if err != nil {
		return "", err
	}

	userInfo := client.URI.Host
	if user := client.URI.User; user != nil && user.Username() != "" {
		userInfo = user.Username() + "@" + userInfo
	}

	return userInfo, err
}
