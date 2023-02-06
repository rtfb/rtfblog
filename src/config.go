package rtfblog

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server
	Notifications
	Interface
}

func (c *Config) getDBConnString() string {
	config := c.Server.DBConn
	if config != "" && config[0] == '$' {
		envVar := os.ExpandEnv(config)
		if envVar == "" {
			logger.Println(fmt.Sprintf("Can't find env var %q", config))
		}
		return envVar
	}
	return config
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
	LogSQL       bool   `yaml:"log_sql"`
	UploadsRoot  string `yaml:"uploads_root"`
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

func hardcodedConf() Config {
	userName := "user"
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error acquiring current user. That can't be good.")
		fmt.Printf("Err = %q", err.Error())
	} else {
		userName = usr.Name
	}
	return Config{
		Server{
			DBConn:       "$RTFBLOG_DB_TEST_URL",
			StaticDir:    "static",
			UploadsRoot:  "build/uploads",
			Port:         ":8080",
			CookieSecret: defaultCookieSecret,
			Log:          "server.log",
			Favicon:      "rtfb.png",
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

func readConfigs() Config {
	homeDir := ""
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error acquiring current user. That can't be good.")
		fmt.Printf("Err = %q", err.Error())
	} else {
		homeDir = usr.HomeDir
	}
	conf := hardcodedConf()
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error from Getwd(), that can't be good: %v\n", err)
		wd = "./"
	}
	// Read the most generic config first, then more specific, each latter will
	// override the former values:
	confPaths := []string{
		"/etc/rtfblogrc",
		filepath.Join(homeDir, ".rtfblogrc"),
		filepath.Join(wd, ".rtfblogrc"),
		filepath.Join(wd, "server.conf"),
	}
	for _, p := range confPaths {
		yml, err := ioutil.ReadFile(p)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		err = yaml.Unmarshal([]byte(yml), &conf)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
	}
	return conf
}
