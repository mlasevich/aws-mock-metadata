package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

// STSManager
type STSManager struct {
	creds Credentials
}

// Constructor for STSManager
func NewSTSManager() STSManager {
	return STSManager{
		creds: NewCredentials(),
	}
}

// Find number of tries to fetch from STS API
func (mngr *STSManager) numberOfTries() int {
	if mngr.creds.NeedsToBeRenewed() {
		if mngr.creds.Expired() {
			return 10
		} else {
			return 1
		}
	}
	return 0
}

// Get credentials
func (mngr *STSManager) GetCredentials(roleArn *string, roleSession *string) (*Credentials, error) {
	tries := mngr.numberOfTries()
	var err error = nil
	if tries == 0 {
		log.Info("Using Cached STS Credentials")
	} else {
		for attempt := 0; attempt < tries; attempt++ {
			log.Debugf("Attempting to refresh Credentials: Attempt %v of %v", attempt+1, tries)
			err = mngr.refreshCredentials(roleArn, roleSession)

			if err == nil {
				log.Debug("Refreshed STS Credentials")
				break
			}
			log.Warnf("Failed to Refresh STS Credentials (Attempt %v of %v)", attempt+1, tries)
		}
	}

	return &mngr.creds, err
}

// Refresh Credentials via STS API
func (mngr *STSManager) refreshCredentials(roleArn *string, roleSession *string) error {
	svc := sts.New(session.New(), &aws.Config{LogLevel: aws.LogLevel(2)})
	resp, err := svc.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         roleArn,
		RoleSessionName: roleSession,
	})

	if err != nil {
		log.Errorf("Error assuming role %+v", err)
		return err
	}

	mngr.creds.Update(resp)
	return nil
}
