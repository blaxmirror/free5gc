package main

import (
	"free5gc/src/app"
	"free5gc/src/etaf/logger"
	"free5gc/src/etaf/service"
	"free5gc/src/etaf/version"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var ETAF = &service.ETAF{}

var appLog *logrus.Entry

func init() {
	appLog = logger.AppLog
}

func main() {
	app := cli.NewApp()
	app.Name = "etaf"
	appLog.Infoln(app.Name)
	appLog.Infoln("ETAF version: ", version.GetVersion())
	app.Usage = "-free5gccfg common configuration file -etafcfg etaf configuration file"
	app.Action = action
	app.Flags = ETAF.GetCliCmd()
	if err := app.Run(os.Args); err != nil {
		logger.AppLog.Errorf("ETAF Run error: %v", err)
	}
}

func action(c *cli.Context) {
	app.AppInitializeWillInitialize(c.String("free5gccfg"))
	ETAF.Initialize(c)
	ETAF.Start()
}
