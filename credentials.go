package main

import (
	"time"

	"github.com/aws/aws-sdk-go/service/sts"
)

// Credentials represent the security credentials metadata response
type Credentials struct {
	Code            string
	LastUpdated     string
	Type            string
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string
	Token           string
	Expiration      string
	Updated         int64 `json:"-"`
	RenewAfter      int64 `json:"-"`
	ExpireAfter     int64 `json:"-"`
}

// Constructor for Credentials
func NewCredentials() Credentials {
	now := time.Now()

	return Credentials{
		Code:        "Success",
		Type:        "AWS-HMAC",
		Expiration:  now.Format("2006-01-02T15:04:05Z"),
		LastUpdated: now.Format("2006-01-02T15:04:05Z"),
		Updated:     now.Unix(),
		RenewAfter:  now.Unix(),
		ExpireAfter: now.Unix(),
	}
}

// Check if Credential is expired
func (creds *Credentials) Expired() bool {
	return time.Now().Unix() > creds.ExpireAfter
}

// Check if Credential needs to be renewed
func (creds *Credentials) NeedsToBeRenewed() bool {
	return time.Now().Unix() > creds.RenewAfter
}

// Update credentials
func (creds *Credentials) Update(assumeRoleOutput *sts.AssumeRoleOutput) {
	now := time.Now()
	expire := assumeRoleOutput.Credentials.Expiration
	renew := now.Unix() + ((expire.Unix() - now.Unix()) / 2)

	creds.AccessKeyID = *assumeRoleOutput.Credentials.AccessKeyId
	creds.SecretAccessKey = *assumeRoleOutput.Credentials.SecretAccessKey
	creds.Token = *assumeRoleOutput.Credentials.SessionToken

	creds.Expiration = expire.Format("2006-01-02T15:04:05Z")
	creds.LastUpdated = now.Format("2006-01-02T15:04:05Z")

	creds.Updated = now.Unix()
	creds.RenewAfter = renew
	creds.ExpireAfter = expire.Unix()
}
