package context

import (
	"fmt"
	"free5gc/lib/idgenerator"
	"free5gc/lib/openapi/models"
	"free5gc/src/etaf/logger"
	"math"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var etafContext = ETAFContext{}
var tmsiGenerator *idgenerator.IDGenerator = nil
var etafUeNGAPIDGenerator *idgenerator.IDGenerator = nil
var etafStatusSubscriptionIDGenerator *idgenerator.IDGenerator = nil

func init() {
	ETAF_Self().LadnPool = make(map[string]*LADN)
	ETAF_Self().EventSubscriptionIDGenerator = idgenerator.NewGenerator(1, math.MaxInt32)
	ETAF_Self().Name = "etaf"
	ETAF_Self().UriScheme = models.UriScheme_HTTPS
	ETAF_Self().RelativeCapacity = 0xff
	ETAF_Self().ServedGuamiList = make([]models.Guami, 0, MaxNumOfServedGuamiList)
	ETAF_Self().PlmnSupportList = make([]PlmnSupportItem, 0, MaxNumOfPLMNs)
	ETAF_Self().NfService = make(map[models.ServiceName]models.NfService)
	ETAF_Self().NetworkName.Full = "free5GC"
	tmsiGenerator = idgenerator.NewGenerator(1, math.MaxInt32)
	etafStatusSubscriptionIDGenerator = idgenerator.NewGenerator(1, math.MaxInt32)
	etafUeNGAPIDGenerator = idgenerator.NewGenerator(1, MaxValueOfEtafUeNgapId)
}

type ETAFContext struct {
	EventSubscriptionIDGenerator    *idgenerator.IDGenerator
	EventSubscriptions              sync.Map
	UePool                          sync.Map         // map[supi]*EtafUe
	RanUePool                       sync.Map         // map[EtafUeNgapID]*RanUe
	EtafRanPool                     sync.Map         // map[net.Conn]*EtafRan
	LadnPool                        map[string]*LADN // dnn as key
	SupportTaiLists                 []models.Tai
	ServedGuamiList                 []models.Guami
	PlmnSupportList                 []PlmnSupportItem
	RelativeCapacity                int64
	NfId                            string
	Name                            string
	NfService                       map[models.ServiceName]models.NfService // nfservice that etaf support
	UriScheme                       models.UriScheme
	BindingIPv4                     string
	SBIPort                         int
	RegisterIPv4                    string
	HttpIPv6Address                 string
	TNLWeightFactor                 int64
	SupportDnnLists                 []string
	ETAFStatusSubscriptions         sync.Map // map[subscriptionID]models.SubscriptionData
	NrfUri                          string
	SecurityAlgorithm               SecurityAlgorithm
	NetworkName                     NetworkName
	NgapIpList                      []string // NGAP Server IP
	T3502Value                      int      // unit is second
	T3512Value                      int      // unit is second
	Non3gppDeregistrationTimerValue int      // unit is second
}

// type ETAFContextEventSubscription struct {
// 	IsAnyUe           bool
// 	IsGroupUe         bool
// 	UeSupiList        []string
// 	Expiry            *time.Time
// 	EventSubscription models.EtafEventSubscription
// }

type PlmnSupportItem struct {
	PlmnId     models.PlmnId   `yaml:"plmnId"`
	SNssaiList []models.Snssai `yaml:"snssaiList,omitempty"`
}

type NetworkName struct {
	Full  string `yaml:"full"`
	Short string `yaml:"short,omitempty"`
}

type SecurityAlgorithm struct {
	IntegrityOrder []uint8 // slice of security.AlgIntegrityXXX
	CipheringOrder []uint8 // slice of security.AlgCipheringXXX
}

func NewPlmnSupportItem() (item PlmnSupportItem) {
	item.SNssaiList = make([]models.Snssai, 0, MaxNumOfSlice)
	return
}

func (context *ETAFContext) TmsiAllocate() int32 {
	tmsi, err := tmsiGenerator.Allocate()
	if err != nil {
		logger.ContextLog.Errorf("Allocate TMSI error: %+v", err)
		return -1
	}
	return int32(tmsi)
}

func (context *ETAFContext) AllocateEtafUeNgapID() (int64, error) {
	return etafUeNGAPIDGenerator.Allocate()
}

func (context *ETAFContext) AllocateGutiToUe(ue *EtafUe) {
	servedGuami := context.ServedGuamiList[0]
	ue.Tmsi = context.TmsiAllocate()

	plmnID := servedGuami.PlmnId.Mcc + servedGuami.PlmnId.Mnc
	tmsiStr := fmt.Sprintf("%08x", ue.Tmsi)
	ue.Guti = plmnID + servedGuami.AmfId + tmsiStr
}

func (context *ETAFContext) AllocateRegistrationArea(ue *EtafUe, anType models.AccessType) {

	// clear the previous registration area if need
	if len(ue.RegistrationArea[anType]) > 0 {
		ue.RegistrationArea[anType] = nil
	}

	// allocate a new tai list as a registration area to ue
	// TODO: algorithm to choose TAI list
	for _, supportTai := range context.SupportTaiLists {
		if reflect.DeepEqual(supportTai, ue.Tai) {
			ue.RegistrationArea[anType] = append(ue.RegistrationArea[anType], supportTai)
			break
		}
	}
}

func (context *ETAFContext) NewETAFStatusSubscription(subscriptionData models.SubscriptionData) (subscriptionID string) {
	id, err := etafStatusSubscriptionIDGenerator.Allocate()
	if err != nil {
		logger.ContextLog.Errorf("Allocate subscriptionID error: %+v", err)
		return ""
	}

	subscriptionID = strconv.Itoa(int(id))
	context.ETAFStatusSubscriptions.Store(subscriptionID, subscriptionData)
	return
}

func (context *ETAFContext) AddEtafUeToUePool(ue *EtafUe, supi string) {
	if len(supi) == 0 {
		logger.ContextLog.Errorf("Supi is nil")
	}
	ue.Supi = supi
	context.UePool.Store(ue.Supi, ue)
}

func (context *ETAFContext) NewEtafUe(supi string) *EtafUe {
	ue := EtafUe{}
	ue.init()

	if supi != "" {
		context.AddEtafUeToUePool(&ue, supi)
	}

	context.AllocateGutiToUe(&ue)

	return &ue
}

func (context *ETAFContext) EtafUeFindByUeContextID(ueContextID string) (*EtafUe, bool) {
	if strings.HasPrefix(ueContextID, "imsi") {
		return context.EtafUeFindBySupi(ueContextID)
	}
	if strings.HasPrefix(ueContextID, "imei") {
		return context.EtafUeFindByPei(ueContextID)
	}
	if strings.HasPrefix(ueContextID, "5g-guti") {
		guti := ueContextID[strings.LastIndex(ueContextID, "-")+1:]
		return context.EtafUeFindByGuti(guti)
	}
	return nil, false
}

func (context *ETAFContext) EtafUeFindBySupi(supi string) (ue *EtafUe, ok bool) {
	if value, loadOk := context.UePool.Load(supi); loadOk {
		ue = value.(*EtafUe)
		ok = loadOk
	}
	return
}

func (context *ETAFContext) EtafUeFindByPei(pei string) (ue *EtafUe, ok bool) {
	context.UePool.Range(func(key, value interface{}) bool {
		candidate := value.(*EtafUe)
		if ok = (candidate.Pei == pei); ok {
			ue = candidate
			return false
		}
		return true
	})
	return
}

func (context *ETAFContext) NewEtafRan(conn net.Conn) *EtafRan {
	ran := EtafRan{}
	ran.SupportedTAList = make([]SupportedTAI, 0, MaxNumOfTAI*MaxNumOfBroadcastPLMNs)
	ran.Conn = conn
	context.EtafRanPool.Store(conn, &ran)
	return &ran
}

// use net.Conn to find RAN context, return *EtafRan and ok bit
func (context *ETAFContext) EtafRanFindByConn(conn net.Conn) (*EtafRan, bool) {
	if value, ok := context.EtafRanPool.Load(conn); ok {
		return value.(*EtafRan), ok
	}
	return nil, false
}

// use ranNodeID to find RAN context, return *EtafRan and ok bit
func (context *ETAFContext) EtafRanFindByRanID(ranNodeID models.GlobalRanNodeId) (*EtafRan, bool) {
	var ran *EtafRan
	var ok bool
	context.EtafRanPool.Range(func(key, value interface{}) bool {
		etafRan := value.(*EtafRan)
		switch etafRan.RanPresent {
		case RanPresentGNbId:
			logger.ContextLog.Infof("aaa: %+v\n", etafRan.RanId.GNbId)
			if etafRan.RanId.GNbId.GNBValue == ranNodeID.GNbId.GNBValue {
				ran = etafRan
				ok = true
				return false
			}
		case RanPresentNgeNbId:
			if etafRan.RanId.NgeNbId == ranNodeID.NgeNbId {
				ran = etafRan
				ok = true
				return false
			}
		case RanPresentN3IwfId:
			if etafRan.RanId.N3IwfId == ranNodeID.N3IwfId {
				ran = etafRan
				ok = true
				return false
			}
		}
		return true
	})
	return ran, ok
}

func (context *ETAFContext) DeleteEtafRan(conn net.Conn) {
	context.EtafRanPool.Delete(conn)
}

func (context *ETAFContext) InSupportDnnList(targetDnn string) bool {
	for _, dnn := range context.SupportDnnLists {
		if dnn == targetDnn {
			return true
		}
	}
	return false
}

func (context *ETAFContext) EtafUeFindByGuti(guti string) (ue *EtafUe, ok bool) {
	context.UePool.Range(func(key, value interface{}) bool {
		candidate := value.(*EtafUe)
		if ok = (candidate.Guti == guti); ok {
			ue = candidate
			return false
		}
		return true
	})
	return
}

func (context *ETAFContext) EtafUeFindByPolicyAssociationID(polAssoId string) (ue *EtafUe, ok bool) {
	context.UePool.Range(func(key, value interface{}) bool {
		candidate := value.(*EtafUe)
		if ok = (candidate.PolicyAssociationId == polAssoId); ok {
			ue = candidate
			return false
		}
		return true
	})
	return
}

func (context *ETAFContext) RanUeFindByEtafUeNgapID(etafUeNgapID int64) *RanUe {
	if value, ok := context.RanUePool.Load(etafUeNgapID); ok {
		return value.(*RanUe)
	} else {
		return nil
	}
}

func (context *ETAFContext) GetIPv4Uri() string {
	return fmt.Sprintf("%s://%s:%d", context.UriScheme, context.RegisterIPv4, context.SBIPort)
}

func (context *ETAFContext) InitNFService(serivceName []string, version string) {
	tmpVersion := strings.Split(version, ".")
	versionUri := "v" + tmpVersion[0]
	for index, nameString := range serivceName {
		name := models.ServiceName(nameString)
		context.NfService[name] = models.NfService{
			ServiceInstanceId: strconv.Itoa(index),
			ServiceName:       name,
			Versions: &[]models.NfServiceVersion{
				{
					ApiFullVersion:  version,
					ApiVersionInUri: versionUri,
				},
			},
			Scheme:          context.UriScheme,
			NfServiceStatus: models.NfServiceStatus_REGISTERED,
			ApiPrefix:       context.GetIPv4Uri(),
			IpEndPoints: &[]models.IpEndPoint{
				{
					Ipv4Address: context.RegisterIPv4,
					Transport:   models.TransportProtocol_TCP,
					Port:        int32(context.SBIPort),
				},
			},
		}
	}
}

// Reset ETAF Context
func (context *ETAFContext) Reset() {
	context.EtafRanPool.Range(func(key, value interface{}) bool {
		context.UePool.Delete(key)
		return true
	})
	for key := range context.LadnPool {
		delete(context.LadnPool, key)
	}
	context.RanUePool.Range(func(key, value interface{}) bool {
		context.RanUePool.Delete(key)
		return true
	})
	context.UePool.Range(func(key, value interface{}) bool {
		context.UePool.Delete(key)
		return true
	})
	// context.EventSubscriptions.Range(func(key, value interface{}) bool {
	// 	context.DeleteEventSubscription(key.(string))
	// 	return true
	// })
	for key := range context.NfService {
		delete(context.NfService, key)
	}
	context.SupportTaiLists = context.SupportTaiLists[:0]
	context.PlmnSupportList = context.PlmnSupportList[:0]
	context.ServedGuamiList = context.ServedGuamiList[:0]
	context.RelativeCapacity = 0xff
	context.NfId = ""
	context.UriScheme = models.UriScheme_HTTPS
	context.SBIPort = 0
	context.BindingIPv4 = ""
	context.RegisterIPv4 = ""
	context.HttpIPv6Address = ""
	context.Name = "etaf"
	context.NrfUri = ""
}

// Create new ETAF context
func ETAF_Self() *ETAFContext {
	return &etafContext
}
