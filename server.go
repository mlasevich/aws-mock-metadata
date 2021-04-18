package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gorilla/mux"
)

var stsManager STSManager = NewSTSManager()

// StartServer starts a newly created http server
func (app *App) StartServer() {
	log.Infof("Listening on port %s:%s", app.AppInterface, app.AppPort)
	if err := http.ListenAndServe(app.AppInterface+":"+app.AppPort, app.NewServer()); err != nil {
		log.Fatalf("Error creating http server: %+v", err)
	}
}

func (app *App) apiVersionPrefixes() []string {
	return []string{"1.0",
		"2007-01-19",
		"2007-03-01",
		"2007-08-29",
		"2007-10-10",
		"2007-12-15",
		"2008-02-01",
		"2008-09-01",
		"2009-04-04",
		"2011-01-01",
		"2011-05-01",
		"2012-01-12",
		"2014-02-25",
		"2014-11-05",
		"2015-10-20",
		"2016-04-19",
		"2016-06-30",
		"2016-09-02",
		"latest",
	}
}

type appHandler func(http.ResponseWriter, *http.Request)

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Infof("Requesting %s", r.RequestURI)
	w.Header().Set("Server", "EC2ws")
	fn(w, r)
}

func (app *App) rootHandler(w http.ResponseWriter, r *http.Request) {
	write(w, strings.Join(app.apiVersionPrefixes(), "\n"))
}

func (app *App) trailingSlashRedirect(w http.ResponseWriter, r *http.Request) {
	location := ""
	if !app.NoSchemeHostRedirects {
		location = "http://169.254.169.254"
	}
	location = fmt.Sprintf("%s%s/", location, r.URL.String())
	w.Header().Set("Location", location)
	w.WriteHeader(301)
}

func (app *App) secondLevelHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `dynamic
meta-data
user-data`)
}

func (app *App) dynamicHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `instance-identity/
`)
}

func (app *App) apiTokenNotPutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "OPTIONS, PUT")
	w.WriteHeader(405)
}

// NOTE: no API methods actually check the X-aws-ec2-metadata-token request header right now...
func (app *App) apiTokenHandler(w http.ResponseWriter, r *http.Request) {
	// Check for X-aws-ec2-metadata-token-ttl-seconds request header
	if r.Header.Get("X-aws-ec2-metadata-token-ttl-seconds") == "" {
		// Not set, 400 Bad Request
		w.WriteHeader(400)
	}

	// Check X-aws-ec2-metadata-token-ttl-seconds is an integer
	seconds_int, err := strconv.Atoi(r.Header.Get("X-aws-ec2-metadata-token-ttl-seconds"))
	if err != nil {
		log.Errorf("apiTokenHandler: Error converting X-aws-ec2-metadata-token-ttl-seconds to integer: %+v", err)
		w.WriteHeader(400)
	}

	// Generate a token, 40 character string, base64 encoded
	token := base64.StdEncoding.EncodeToString([]byte(RandStringBytesMaskImprSrc(40)))

	w.Header().Set("X-Aws-Ec2-Metadata-Token-Ttl-Seconds", strconv.Itoa(seconds_int))
	write(w, token)
}

func (app *App) instanceIdentityHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `document
pkcs7
signature
`)
}

type InstanceIdentityDocument struct {
	InstanceId         string  `json:"instanceId"`
	BillingProducts    *string `json:"billingProducts"`
	ImageId            string  `json:"imageId"`
	Architecture       string  `json:"architecture"`
	PendingTime        string  `json:"pendingTime"`
	InstanceType       string  `json:"instanceType"`
	AccountId          string  `json:"accountId"`
	KernelId           *string `json:"kernelId"`
	RamdiskId          *string `json:"ramdiskId"`
	Region             string  `json:"region"`
	Version            string  `json:"version"`
	AvailabilityZone   string  `json:"availabilityZone"`
	DevpayProductCodes *string `json:"devpayProductCodes"`
	PrivateIp          string  `json:"privateIp"`
}

func (app *App) instanceIdentityDocumentHandler(w http.ResponseWriter, r *http.Request) {
	document := InstanceIdentityDocument{
		AvailabilityZone:   app.AvailabilityZone,
		Region:             app.AvailabilityZone[:len(app.AvailabilityZone)-1],
		DevpayProductCodes: nil,
		PrivateIp:          app.PrivateIp,
		Version:            "2010-08-31",
		InstanceId:         app.InstanceID,
		BillingProducts:    nil,
		InstanceType:       app.InstanceType,
		AccountId:          app.AccountID,
		ImageId:            app.AmiID,
		PendingTime:        "2016-04-15T12:14:15Z",
		Architecture:       "x86_64",
		KernelId:           nil,
		RamdiskId:          nil,
	}
	result, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		log.Errorf("Error marshalling json %+v", err)
		http.Error(w, err.Error(), 500)
	}
	write(w, string(result))
}

// We cannot impersonate AWS and generate matching signatures here.
// Just return placeholder data instead.
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
func (app *App) instanceIdentityPkcs7Handler(w http.ResponseWriter, r *http.Request) {
	write(w, `PKCS7`)
}

// We cannot impersonate AWS and generate matching signatures here.
// Just return placeholder data instead.
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
func (app *App) instanceIdentitySignatureHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `SIGNATURE`)
}

func (app *App) metaDataHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: if IAM Role/Instance Profile is disabled, don't add iam/ to the list (same behavior as real metadata service)
	write(w, `ami-id
ami-launch-index
ami-manifest-path
block-device-mapping/
hostname
iam/
instance-action
instance-id
instance-type
local-hostname
local-ipv4
mac
metrics/
network/
placement/
profile
public-hostname
public-ipv4
reservation-id
security-groups
services/`)
}

func (app *App) amiIdHandler(w http.ResponseWriter, r *http.Request) {
	write(w, app.AmiID)
}

func (app *App) amiLaunchIndexHandler(w http.ResponseWriter, r *http.Request) {
	write(w, "0")
}

func (app *App) amiManifestPathHandler(w http.ResponseWriter, r *http.Request) {
	write(w, "(unknown)")
}

func (app *App) blockDeviceMappingHandler(w http.ResponseWriter, r *http.Request) {
	// Not exposing any extra volumes for now, this is pretty standard for an EBS backed EC2 instance.
	write(w, `ami
root`)
}

func (app *App) blockDeviceMappingAmiHandler(w http.ResponseWriter, r *http.Request) {
	write(w, "/dev/xvda")
}

func (app *App) blockDeviceMappingRootHandler(w http.ResponseWriter, r *http.Request) {
	write(w, "/dev/xvda")
}

func (app *App) hostnameHandler(w http.ResponseWriter, r *http.Request) {
	write(w, app.Hostname)
}

func (app *App) iamHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `info
security-credentials/`)
}

func (app *App) infoHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `{
  "Code" : "Success",
  "LastUpdated" : "2018-02-26T23:50:00Z",
  "InstanceProfileArn" : "arn:aws:iam::123456789012:instance-profile/some-instance-profile",
  "InstanceProfileId" : "some-instance-profile-id"
}`)
}

func (app *App) instanceActionHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `none`)
}

func (app *App) instanceIDHandler(w http.ResponseWriter, r *http.Request) {
	write(w, app.InstanceID)
}

func (app *App) instanceTypeHandler(w http.ResponseWriter, r *http.Request) {
	write(w, app.InstanceType)
}

func (app *App) localHostnameHandler(w http.ResponseWriter, r *http.Request) {
	write(w, app.Hostname)
}

func (app *App) privateIpHandler(w http.ResponseWriter, r *http.Request) {
	write(w, app.PrivateIp)
}

func (app *App) macHandler(w http.ResponseWriter, r *http.Request) {
	write(w, app.MacAddress)
}

func (app *App) metricsHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `vhostmd`)
}

func (app *App) metricsVhostmdHandler(w http.ResponseWriter, r *http.Request) {
	// No idea what actually lives here right now, leaving as a placeholder.
	write(w, `<?xml version="1.0" encoding="UTF-8"?>`)
}

func (app *App) networkHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `interfaces/`)
}

func (app *App) networkInterfacesHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `macs/`)
}

func (app *App) availabilityZoneHandler(w http.ResponseWriter, r *http.Request) {
	write(w, app.AvailabilityZone)
}

func (app *App) regionHandler(w http.ResponseWriter, r *http.Request) {
	write(w, app.AvailabilityZone[:len(app.AvailabilityZone)-1])
}

func (app *App) securityCredentialsHandler(w http.ResponseWriter, r *http.Request) {
	write(w, app.RoleName)
}

func (app *App) networkInterfacesMacsHandler(w http.ResponseWriter, r *http.Request) {
	write(w, app.MacAddress+"/")
}

func (app *App) networkInterfacesMacsAddrHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `device-number
interface-id
ipv4-associations/
local-hostname
local-ipv4s
mac
owner-id
public-hostname
public-ipv4s
security-group-ids
security-groups
subnet-id
subnet-ipv4-cidr-block
vpc-id
vpc-ipv4-cidr-block
vpc-ipv4-cidr-blocks`)
}

func (app *App) nimAddrDeviceNumberHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `0`)
}

func (app *App) nimAddrInterfaceIdHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `eni-asdfasdf`)
}

func (app *App) profileHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `default-hvm`)
}

func (app *App) vpcHandler(w http.ResponseWriter, r *http.Request) {
	write(w, app.VpcID)
}

func (app *App) placementHandler(w http.ResponseWriter, r *http.Request) {
	write(w, `

`)
}

func (app *App) mockRoleHandler(w http.ResponseWriter, r *http.Request) {
	// TODOLATER: round to nearest hour, to ensure test coverage passes more reliably?
	now := time.Now().UTC()
	expire := now.Add(6 * time.Hour)
	format := "2006-01-02T15:04:05Z"
	write(w, fmt.Sprintf(`{
  "Code" : "Success",
  "LastUpdated" : "%s",
  "Type" : "AWS-HMAC",
  "AccessKeyId" : "mock-access-key-id",
  "SecretAccessKey" : "mock-secret-access-key",
  "Token" : "mock-token",
  "Expiration" : "%s"
}`, now.Format(format), expire.Format(format)))
}

// Handle Role Credential Request
func (app *App) roleHandler(w http.ResponseWriter, r *http.Request) {
	credentials, err := stsManager.GetCredentials(
		aws.String(app.RoleArn), aws.String(app.Hostname),
	)
	if err != nil {
		log.Errorf("Error Getting credentials %+v", err)
		http.Error(w, err.Error(), 500)
	} else {
		if err := json.NewEncoder(w).Encode(credentials); err != nil {
			log.Errorf("Error sending json %+v", err)
			http.Error(w, err.Error(), 500)
		}
	}
}

func (app *App) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]
	w.WriteHeader(404)
	write(w, `<?xml version="1.0" encoding="iso-8859-1"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en" lang="en">
<head>
<title>404 - Not Found</title>
</head>
<body>
<h1>404 - Not Found</h1>
</body>
</html>`)
	log.Errorf("Not found " + path)
}

func write(w http.ResponseWriter, s string) {
	if _, err := w.Write([]byte(s)); err != nil {
		log.Errorf("Error writing response: %+v", err)
	}
}
