package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/pflag"
)

var Version = "dev"
var Program = "aws-mock-metadata"

// App encapsulates all of the parameters necessary for starting up
// an aws mock metadata server. These can either be set via command line or directly.
type App struct {
	AmiID            string `json:"ami-id"`
	AvailabilityZone string `json:"availability-zone"`
	AppInterface     string `json:"bind"`
	AppPort          string `json:"port"`
	Hostname         string `json:"hostname"`
	InstanceID       string `json:"instance-id"`
	AccountID        string `json:"account-id"`
	InstanceType     string `json:"instance-type"`
	MacAddress       string `json:"mac-address"`
	PrivateIp        string `json:"private-ip"`
	// If set, will return mocked credentials to the IAM instance profile instead of using STS to retrieve real credentials.
	MockInstanceProfile   bool   `json:"mock-instance-profile"`
	RoleArn               string `json:"role-arn"`
	RoleName              string `json:"role-name"`
	ShowVersion           bool
	Verbose               bool   `json:"verbose"`
	VpcID                 string `json:"vpc-id"`
	NoSchemeHostRedirects bool   `json:"no-scheme-redirects"`
	ConfigFile            string `json:"config"`
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	app := &App{}
	app.addFlags(pflag.CommandLine)
	pflag.Parse()
	if app.ShowVersion {
		fmt.Printf("%s v%s\n", Program, Version)
		os.Exit(0)
	}
	app.loadFromFile()
	if app.Verbose {
		log.SetLevel(log.DebugLevel)
	}
	conf, _ := json.MarshalIndent(app, "", "  ")

	log.Debugf("App config is: \n%s", conf)
	app.StartServer()
}

func (app *App) addFlags(fs *pflag.FlagSet) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	mac, ip, err := getNetMapAndIP()
	if err != nil {
		mac = "UN:KN:OW:NM:AC:AD"
		ip = "127.0.0.2"
	}

	fs.StringVar(&app.AmiID, "ami-id", "ami-default", "EC2 Instance AMI ID")
	fs.StringVar(&app.AvailabilityZone, "availability-zone", "us-west-2", "Availability Zone")
	fs.StringVar(&app.AppInterface, "app-interface", "169.254.169.254", "HTTP Network Interface")
	fs.StringVar(&app.AppPort, "app-port", "80", "HTTP Port")
	fs.StringVar(&app.Hostname, "hostname", hostname, "EC2 Instance Hostname")
	fs.StringVar(&app.InstanceID, "instance-id", hostname, "EC2 Instance ID")
	fs.StringVar(&app.InstanceType, "instance-type", "x1.unknown", "EC2 Instance Type")
	fs.StringVar(&app.AccountID, "account-id", "123456789", "AWS Account ID")
	fs.StringVar(&app.MacAddress, "mac-address", mac, "ENI MAC Address")
	fs.StringVar(&app.PrivateIp, "private-ip", ip, "ENI Private IP")
	fs.BoolVar(&app.MockInstanceProfile, "mock-instance-profile", false, "Use mocked IAM Instance Profile credentials (instead of STS generated credentials)")
	fs.StringVar(&app.RoleArn, "role-arn", app.RoleArn, "IAM Role ARN")
	fs.StringVar(&app.RoleName, "role-name", app.RoleName, "IAM Role Name")
	fs.BoolVarP(&app.ShowVersion, "version", "v", false, "Show version and exit")
	fs.BoolVar(&app.Verbose, "verbose", false, "Verbose")
	fs.StringVar(&app.VpcID, "vpc-id", "vpc-unknown", "VPC ID")
	fs.BoolVar(&app.NoSchemeHostRedirects, "no-scheme-host-redirects", app.NoSchemeHostRedirects, "Disable the scheme://host prefix in Location redirect headers")
	fs.StringVar(&app.ConfigFile, "config", "/etc/aws-ec2-metadata/metadata.json", "File to load configuration from")
}

func (app *App) loadFromFile() {
	log.Debugf("Loading config from file %s", app.ConfigFile)
	data, err := ioutil.ReadFile(app.ConfigFile)
	if err != nil {
		log.Infof("Failed to load conf: %+v", err)
		return
	}
	err = json.Unmarshal([]byte(data), &app)
	if err != nil {
		log.Infof("Failed to parse conf file %s: %s", app.ConfigFile, err)
	}
}

func getNetMapAndIP() (string, string, error) {
	interfaces, err := net.Interfaces()
	var mac string
	var ip string

	if err != nil {
		return mac, ip, err
	}

	for _, iface := range interfaces {
		if (iface.Flags & net.FlagLoopback) == 0 {
			if iface.Flags&net.FlagUp == 0 {
				continue // interface down
			}
			if iface.Flags&net.FlagLoopback != 0 {
				continue // loopback interface
			}
			m := iface.HardwareAddr.String()

			if m != "" {
				mac = m
				addrs, err := iface.Addrs()
				if err == nil {
					var ipv4Addr net.IP
					for _, addr := range addrs {
						if ipv4Addr = addr.(*net.IPNet).IP.To4(); ipv4Addr != nil {
							break
						}
					}
					if ipv4Addr != nil {
						ip = ipv4Addr.String()
					} else {
						ip = "127.0.0.1"
					}
				}
			}
		}
	}
	return mac, ip, nil
}
