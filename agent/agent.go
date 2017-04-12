package agent

import (
	"github.com/spf13/viper"
	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
	"github.com/foofilers/confHubCli/rest"
	"io/ioutil"
)

type ConfHubConfig struct {
	Url      string
	Username string
	Password string
}

type AppConfig struct {
	Name       string
	Version    string
	ConfPath   string
	Format     string
	Username   string
	Password   string
	confClient *rest.ConfHubClient
}

var AppConfigs []AppConfig
var ConfHubServerConfig *ConfHubConfig

func ReadConfiguration() {
	ConfHubServerConfig = &ConfHubConfig{}
	if err := viper.UnmarshalKey("confHub", ConfHubServerConfig); err != nil {
		logrus.Fatal(err)
	}
	// loading apps
	apps := viper.Get("apps")
	logrus.Debug(apps)
	appLst := apps.([]interface{})
	AppConfigs = make([]AppConfig, len(appLst), len(appLst))
	for i, app := range appLst {
		AppConfigs[i] = AppConfig{}
		if err := mapstructure.Decode(app, &AppConfigs[i]); err != nil {
			logrus.Fatal(err)
		}
		logrus.Debugf("%+v", AppConfigs[i])
	}
}

func UpdateAllConfiguration() {
	for _, app := range AppConfigs {
		if err := app.UpdateConfiguration(); err != nil {
			logrus.Error(err)
		}
	}
}

func (appCnf *AppConfig) ConfHubClient() *rest.ConfHubClient {
	if appCnf.confClient == nil {
		username := appCnf.Username
		password := appCnf.Username
		if len(username) == 0 {
			username = ConfHubServerConfig.Username
		}
		if len(password) == 0 {
			password = ConfHubServerConfig.Password
		}
		appCnf.confClient = rest.NewConfHubClient(ConfHubServerConfig.Url, username, password)
	}
	return appCnf.confClient
}

/**
	Update the configuration to file
 */
func (appCnf *AppConfig) UpdateConfiguration() error {
	logrus.Infof("Updating configuration %+v", appCnf)
	cnf, err := appCnf.ConfHubClient().GetFormattedConfigs(appCnf.Name, appCnf.Version, appCnf.Format)
	if err != nil {
		return err
	}
	logrus.Debugf("Retreived configuration for %+v\n%s", appCnf, cnf)
	err = ioutil.WriteFile(appCnf.ConfPath, []byte(cnf), 0664)
	logrus.Infof("Configuration %+s updated", appCnf)
	return err
}
