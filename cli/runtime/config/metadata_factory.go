// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	configapi "github.com/vmware-tanzu/tanzu-framework/cli/runtime/apis/config/v1alpha1"
	"github.com/vmware-tanzu/tanzu-framework/cli/runtime/config/nodeutils"
)

// GetMetadataNode retrieves the config from the local directory without acquiring the lock
func GetMetadataNode() (*yaml.Node, error) {
	// Acquire tanzu config lock
	AcquireTanzuMetadataLock()
	defer ReleaseTanzuMetadataLock()
	return GetMetadataNodeNoLock()
}

// GetMetadataNodeNoLock retrieves the config from the local directory without acquiring the lock
func GetMetadataNodeNoLock() (*yaml.Node, error) {
	cfgPath, err := MetadataFilePath()
	if err != nil {
		return nil, errors.Wrap(err, "failed getting config metadata path")
	}

	bytes, err := os.ReadFile(cfgPath)
	if err != nil || len(bytes) == 0 {
		node, err := NewMetadataNode()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create new config metadata")
		}
		return node, nil
	}
	var node yaml.Node

	err = yaml.Unmarshal(bytes, &node)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct struct from config metadata data")
	}
	node.Content[0].Style = 0

	return &node, nil
}

func NewMetadataNode() (*yaml.Node, error) {
	c := newMetadataNode()
	node, err := nodeutils.ConvertToNode[configapi.Metadata](c)
	node.Content[0].Style = 0
	if err != nil {
		return nil, err
	}
	return node, nil
}

func newMetadataNode() *configapi.Metadata {
	c := &configapi.Metadata{}
	// Check if the lock is acquired by the current process or not
	// If not try to acquire the lock before Storing the client config
	// and release the lock after updating the config
	if !IsTanzuMetadataLockAcquired() {
		AcquireTanzuMetadataLock()
		defer ReleaseTanzuMetadataLock()
	}
	return c
}

func persistConfigMetadata(node *yaml.Node) error {
	path, err := MetadataFilePath()
	if err != nil {
		return errors.Wrap(err, "could not find config metadata path")
	}
	return persistNode(node, WithCfgPath(path))
}
