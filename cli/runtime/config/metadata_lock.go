// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/juju/fslock"
)

const (
	LocalTanzuMetadataFileLock = ".tanzu-metadata.lock"
	// DefaultMetadataLockTimeout is the default time waiting on the filelock
	DefaultMetadataLockTimeout = 10 * time.Minute
)

var tanzuMetadataLockFile string

// tanzuMetadataLock used as a static lock variable that stores fslock
// This is used for interprocess locking of the config file
var tanzuMetadataLock *fslock.Lock

// mutexMetadata is used to handle the locking behavior between concurrent calls
// within the existing process trying to acquire the lock
var mutexMetadata sync.Mutex

// AcquireTanzuMetadataLock tries to acquire lock to update tanzu config file with timeout
func AcquireTanzuMetadataLock() {
	var err error

	if tanzuMetadataLockFile == "" {
		path, err := MetadataFilePath()
		if err != nil {
			panic(fmt.Sprintf("cannot get config path while acquiring lock on tanzu config file, reason: %v", err))
		}
		tanzuMetadataLockFile = filepath.Join(filepath.Dir(path), LocalTanzuMetadataFileLock)
	}

	// using fslock to handle interprocess locking
	lock, err := getFileLockWithTimeOut(tanzuMetadataLockFile, DefaultMetadataLockTimeout)
	if err != nil {
		panic(fmt.Sprintf("cannot acquire lock for tanzu config file, reason: %v", err))
	}

	// Lock the mutex to prevent concurrent calls to acquire and configure the tanzuConfigLock
	mutexMetadata.Lock()
	tanzuMetadataLock = lock
}

// ReleaseTanzuMetadataLock releases the lock if the tanzuConfigLock was acquired
func ReleaseTanzuMetadataLock() {
	if tanzuMetadataLock == nil {
		return
	}
	if errUnlock := tanzuMetadataLock.Unlock(); errUnlock != nil {
		panic(fmt.Sprintf("cannot release lock for tanzu config file, reason: %v", errUnlock))
	}

	tanzuMetadataLock = nil
	// Unlock the mutex to allow other concurrent calls to axcquire and configure the tanzuConfigLock
	mutexMetadata.Unlock()
}

// IsTanzuMetadataLockAcquired checks the lock status and returns
// true if the lock is acquired by the current process or returns
// false otherwise
func IsTanzuMetadataLockAcquired() bool {
	return tanzuMetadataLock != nil
}
