// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.

package core

import (
	"github.com/wercker/wercker/util"
	ocisdkcomm "github.com/oracle/oci-go-sdk/common"
	ocisdkstorage "github.com/oracle/oci-go-sdk/objectstorage"
	"os"
	"context"
)

/*
OciEnvVarPrefix is the prefix to use for all environment variables needed by the OCI SDK
 */
const OciEnvVarPrefix  = "wkr"

// NewS3Store creates a new S3Store
func NewOciStore(options *OciOptions) *OciStore {
	logger := util.RootLogger().WithField("Logger", "OciStore")
	if options == nil {
		logger.Panic("options cannot be nil")
	}

	return &OciStore {
		logger:  logger,
		options: options,
	}
}

// S3Store stores files in S3
type OciStore struct {
	logger  *util.LogEntry
	options *OciOptions
}

// StoreFromFile copies the file from args.Path to options.Bucket + args.Key.
func (this *OciStore) StoreFromFile(args *StoreFromFileArgs) error {
	if args.MaxTries == 0 {
		args.MaxTries = 1
	}
	configProv := ocisdkcomm.ConfigurationProviderEnvironmentVariables(OciEnvVarPrefix, "");
	objStoreClient, err := ocisdkstorage.NewObjectStorageClientWithConfigurationProvider(configProv)
	if err != nil {
		return err
	}
	this.logger.WithFields(util.LogFields{
		"Bucket":   this.options.Bucket,
		"Name":     args.Key,
		"Path":     args.Path,
		"Namepace":   this.options.Namespace,
	}).Info("Uploading file to OCI ObjectStore")

	fileInfo, err := os.Stat(args.Path)
	contentLength := int(fileInfo.Size()) //OCI SDK requires int content length
	file, err := os.Open(args.Path)
	if err != nil {
		this.logger.WithField("Error", err).Error("Unable to open input file")
		return err
	}
	defer file.Close()

	putRequest := ocisdkstorage.PutObjectRequest{
		NamespaceName:      &this.options.Namespace,
		BucketName:         &this.options.Bucket,
		ObjectName:         &args.Key,
		PutObjectBody:      file,
		ContentLength:      &contentLength,
	}
	if err != nil {
		return err
	}
	resp, err := objStoreClient.PutObject(context.Background(), putRequest)
	if err != nil {
		return err
	}
	this.logger.Debugf("Completed put object %s in bucket %s. Response from server is: %s",
		this.options.Namespace, this.options.Bucket, resp)
	return nil
}
