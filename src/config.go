package main

import (
	"fmt"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

var (
	conf *Config
)

type Config struct {
	Server
	Notifications
	Interface
}

type Server struct {
	DBConn       string `yaml:"db_conn"`
	StaticDir    string `yaml:"static_dir"`
	Favicon      string
	Port         string
	TLSPort      string `yaml:"tls_port"`
	TLSCert      string `yaml:"tls_cert"`
	TLSKey       string `yaml:"tls_key"`
	CookieSecret string `yaml:"cookie_secret"`
	Log          string
}

type Notifications struct {
	SendEmail    bool   `yaml:"send_email"`
	SenderAcct   string `yaml:"sender_acct"`
	SenderPasswd string `yaml:"sender_passwd"`
	AdminEmail   string `yaml:"admin_email"`
}

type Interface struct {
	BlogTitle string `yaml:"blog_title"`
	BlogDescr string `yaml:"blog_descr"`
	Language  string
}

func hardcodedConf() *Config {
	userName := "user"
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error acquiring current user. That can't be good.")
		fmt.Printf("Err = %q", err.Error())
	} else {
		userName = usr.Name
	}
	return &Config{
		Server{
			DBConn:       "$RTFBLOG_DB_TEST_URL",
			StaticDir:    "static",
			Port:         ":8080",
			CookieSecret: defaultCookieSecret,
			Log:          "server.log",
		},
		Notifications{
			SendEmail: false,
		},
		Interface{
			BlogTitle: fmt.Sprintf("%s's blog", userName),
			BlogDescr: "Blogity blog blog",
			Language:  "en-US",
		},
	}
}

func readConfigs(assets *AssetBin) *Config {
	homeDir := ""
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error acquiring current user. That can't be good.")
		fmt.Printf("Err = %q", err.Error())
	} else {
		homeDir = usr.HomeDir
	}
	conf := hardcodedConf()
	// Read the most generic config first, then more specific, each latter will
	// override the former values:
	confPaths := []string{
		"/etc/rtfblogrc",
		filepath.Join(homeDir, ".rtfblogrc"),
		".rtfblogrc",
		"server.conf",
	}
	for _, p := range confPaths {
		yml, err := assets.Load(p)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		err = yaml.Unmarshal([]byte(yml), conf)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
	}
	return conf
}
