//+build !debug

package util

import (
	"free5gc/lib/path_util"
)

var EtafLogPath = path_util.Gofree5gcPath("free5gc/etafsslkey.log")
var EtafPemPath = path_util.Gofree5gcPath("free5gc/support/TLS/etaf.pem")
var EtafKeyPath = path_util.Gofree5gcPath("free5gc/support/TLS/etaf.key")
var DefaultEtafConfigPath = path_util.Gofree5gcPath("free5gc/config/etafcfg.conf")
