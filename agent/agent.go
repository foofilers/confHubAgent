package agent

import (
	"github.com/spf13/viper"
	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
	"github.com/foofilers/confHubCli/rest"
	"io/ioutil"
	"sync"
	"time"
	"os"
)

type ConfHubConfig struct {
	Url      string
	Username string
	Password string
}

type AppConfig struct {
	Name            string
	Version         string
	ConfPath        string
	Format          string
	Username        string
	Password        string
	Permission      os.FileMode
	confClient      *rest.ConfHubClient
	updateConfMutex sync.Mutex
}

var AppConfigs []*AppConfig
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
	AppConfigs = make([]*AppConfig, len(appLst), len(appLst))
	for i, app := range appLst {
		//defaults
		AppConfigs[i] = &AppConfig{
			Permission:0664,
		}
		if err := mapstructure.Decode(app, AppConfigs[i]); err != nil {
			logrus.Fatal(err)
		}
		logrus.Debugf("Loaded configuration: %+v", AppConfigs[i])
	}

	logrus.Debugf("full configs: %+v", AppConfigs)
}

func UpdateAllConfiguration() {
	for _, app := range AppConfigs {
		logrus.Debugf("Update all config for app:%+v", app)
		if err := app.UpdateConfiguration(); err != nil {
			logrus.Error(err)
		}
	}
}

func WatchingApplications() {
	for _, app := range AppConfigs {
		logrus.Debugf("watch config for app:%+v", app)
		go func(a *AppConfig) {
			if err := a.WatchChanges(); err != nil {
				logrus.Fatal(err)
			}
		}(app)
	}
	for {
		//waiting forever
		time.Sleep(1 * time.Second)
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
	appCnf.updateConfMutex.Lock()
	defer appCnf.updateConfMutex.Unlock()
	logrus.Infof("Updating configuration %+v", appCnf)
	cnf, err := appCnf.ConfHubClient().GetFormattedConfigs(appCnf.Name, appCnf.Version, appCnf.Format)
	if err != nil {
		return err
	}
	logrus.Debugf("Retreived configuration for %+v\n%s", appCnf, cnf)
	err = ioutil.WriteFile(appCnf.ConfPath, []byte(cnf), appCnf.Permission)
	logrus.Infof("Configuration %+s updated", appCnf)
	return err
}

func (appCnf *AppConfig) WatchChanges() error {
	logrus.Infof("Start watching changes for application %+v", appCnf)
	watchCh, err := appCnf.ConfHubClient().WatchApp([]string{appCnf.Name})
	if err != nil {
		return err
	}

	for ch := range watchCh {
		if ch.Application != appCnf.Name {
			logrus.Errorf("Received a change notification for %s application but I'm %s", ch.Application, appCnf.Name)
		} else {
			logrus.Debugf("Application %s changed", appCnf.Name)
			appCnf.UpdateConfiguration()
		}
	}
	return nil
}