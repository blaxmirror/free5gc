package service

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gin-contrib/cors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"free5gc/lib/MongoDBLibrary"
	"free5gc/lib/http2_util"
	"free5gc/lib/logger_util"
	"free5gc/lib/openapi/models"
	"free5gc/lib/path_util"
	"free5gc/src/app"
	"free5gc/src/etaf/consumer"
	"free5gc/src/etaf/context"
	"free5gc/src/etaf/factory"
	"free5gc/src/etaf/httpcallback"
	"free5gc/src/etaf/logger"

	// ngap_message "free5gc/src/etaf/ngap/message"
	// ngap_service "free5gc/src/etaf/ngap/service"
	"free5gc/src/etaf/oam"
	"free5gc/src/etaf/tracking"
	"free5gc/src/etaf/util"
)

type ETAF struct{}

type (
	// Config information.
	Config struct {
		etafcfg string
	}
)

var config Config

var etafCLi = []cli.Flag{
	cli.StringFlag{
		Name:  "free5gccfg",
		Usage: "common config file",
	},
	cli.StringFlag{
		Name:  "etafcfg",
		Usage: "etaf config file",
	},
}

var initLog *logrus.Entry

func init() {
	initLog = logger.InitLog
}

func (*ETAF) GetCliCmd() (flags []cli.Flag) {
	return etafCLi
}

func (*ETAF) Initialize(c *cli.Context) {

	config = Config{
		etafcfg: c.String("etafcfg"),
	}

	if config.etafcfg != "" {
		factory.InitConfigFactory(config.etafcfg)
	} else {
		DefaultEtafConfigPath := path_util.Gofree5gcPath("free5gc/config/etafcfg.conf")
		factory.InitConfigFactory(DefaultEtafConfigPath)
	}

	if app.ContextSelf().Logger.ETAF.DebugLevel != "" {
		level, err := logrus.ParseLevel(app.ContextSelf().Logger.ETAF.DebugLevel)
		if err != nil {
			initLog.Warnf("Log level [%s] is not valid, set to [info] level", app.ContextSelf().Logger.ETAF.DebugLevel)
			logger.SetLogLevel(logrus.InfoLevel)
		} else {
			logger.SetLogLevel(level)
			initLog.Infof("Log level is set to [%s] level", level)
		}
	} else {
		initLog.Infoln("Log level is default set to [info] level")
		logger.SetLogLevel(logrus.InfoLevel)
	}

	logger.SetReportCaller(app.ContextSelf().Logger.ETAF.ReportCaller)

}

func (etaf *ETAF) FilterCli(c *cli.Context) (args []string) {
	for _, flag := range etaf.GetCliCmd() {
		name := flag.GetName()
		value := fmt.Sprint(c.Generic(name))
		if value == "" {
			continue
		}

		args = append(args, "--"+name, value)
	}
	return args
}

func (etaf *ETAF) Start() {

	MongoDBLibrary.SetMongoDB(factory.EtafConfig.Configuration.MongoDBName, factory.EtafConfig.Configuration.MongoDBUrl)

	initLog.Infoln("Server started")

	router := logger_util.NewGinWithLogrus(logger.GinLog)
	router.Use(cors.New(cors.Config{
		AllowMethods: []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE"},
		AllowHeaders: []string{"Origin", "Content-Length", "Content-Type", "User-Agent", "Referrer", "Host",
			"Token", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowAllOrigins:  true,
		MaxAge:           86400,
	}))

	httpcallback.AddService(router)
	oam.AddService(router)
	for _, serviceName := range factory.EtafConfig.Configuration.ServiceNameList {
		switch models.ServiceName(serviceName) {
		case models.ServiceName_NETAF_TRACK:
			tracking.AddService(router)
		}
	}

	self := context.ETAF_Self()
	util.InitEtafContext(self)

	addr := fmt.Sprintf("%s:%d", self.BindingIPv4, self.SBIPort)

	// ngap_service.Run(self.NgapIpList, 38412, ngap.Dispatch)

	// Register to NRF
	var profile models.NfProfile
	if profileTmp, err := consumer.BuildNFInstance(self); err != nil {
		initLog.Error("Build ETAF Profile Error")
	} else {
		profile = profileTmp
	}

	if _, nfId, err := consumer.SendRegisterNFInstance(self.NrfUri, self.NfId, profile); err != nil {
		initLog.Warnf("Send Register NF Instance failed: %+v", err)
	} else {
		self.NfId = nfId
	}

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChannel
		etaf.Terminate()
		os.Exit(0)
	}()

	server, err := http2_util.NewServer(addr, util.EtafLogPath, router)

	if server == nil {
		initLog.Errorf("Initialize HTTP server failed: %+v", err)
		return
	}

	if err != nil {
		initLog.Warnf("Initialize HTTP server: %+v", err)
	}

	serverScheme := factory.EtafConfig.Configuration.Sbi.Scheme
	if serverScheme == "http" {
		err = server.ListenAndServe()
	} else if serverScheme == "https" {
		err = server.ListenAndServeTLS(util.EtafPemPath, util.EtafKeyPath)
	}

	if err != nil {
		initLog.Fatalf("HTTP server setup failed: %+v", err)
	}
}

func (etaf *ETAF) Exec(c *cli.Context) error {

	//ETAF.Initialize(cfgPath, c)

	initLog.Traceln("args:", c.String("etafcfg"))
	args := etaf.FilterCli(c)
	initLog.Traceln("filter: ", args)
	command := exec.Command("./etaf", args...)

	stdout, err := command.StdoutPipe()
	if err != nil {
		initLog.Fatalln(err)
	}
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		in := bufio.NewScanner(stdout)
		for in.Scan() {
			fmt.Println(in.Text())
		}
		wg.Done()
	}()

	stderr, err := command.StderrPipe()
	if err != nil {
		initLog.Fatalln(err)
	}
	go func() {
		in := bufio.NewScanner(stderr)
		for in.Scan() {
			fmt.Println(in.Text())
		}
		wg.Done()
	}()

	go func() {
		if err = command.Start(); err != nil {
			initLog.Errorf("ETAF Start error: %+v", err)
		}
		wg.Done()
	}()

	wg.Wait()

	return err
}

// Used in ETAF planned removal procedure
func (etaf *ETAF) Terminate() {
	logger.InitLog.Infof("Terminating ETAF...")
	// etafSelf := context.ETAF_Self()

	// TODO: forward registered UE contexts to target ETAF in the same ETAF set if there is one

	// deregister with NRF
	problemDetails, err := consumer.SendDeregisterNFInstance()
	if problemDetails != nil {
		logger.InitLog.Errorf("Deregister NF instance Failed Problem[%+v]", problemDetails)
	} else if err != nil {
		logger.InitLog.Errorf("Deregister NF instance Error[%+v]", err)
	} else {
		logger.InitLog.Infof("[ETAF] Deregister from NRF successfully")
	}

	// send ETAF status indication to ran to notify ran that this ETAF will be unavailable
	logger.InitLog.Infof("Send ETAF Status Indication to Notify RANs due to ETAF terminating")
	// unavailableGuamiList := ngap_message.BuildUnavailableGUAMIList(etafSelf.ServedGuamiList)
	// etafSelf.EtafRanPool.Range(func(key, value interface{}) bool {
	// 	ran := value.(*context.EtafRan)
	// 	ngap_message.SendETAFStatusIndication(ran, unavailableGuamiList)
	// 	return true
	// })

	// ngap_service.Stop()

	logger.InitLog.Infof("ETAF terminated")
}
