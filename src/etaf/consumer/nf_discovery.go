package consumer

import (
	"context"
	"fmt"
	"free5gc/lib/openapi/Nnrf_NFDiscovery"
	"free5gc/lib/openapi/models"
	etaf_context "free5gc/src/etaf/context"
	"free5gc/src/etaf/logger"
	"free5gc/src/etaf/util"
	"net/http"
)

func SendSearchNFInstances(nrfUri string, targetNfType, requestNfType models.NfType,
	param *Nnrf_NFDiscovery.SearchNFInstancesParamOpts) (models.SearchResult, error) {

	// Set client and set url
	configuration := Nnrf_NFDiscovery.NewConfiguration()
	configuration.SetBasePath(nrfUri)
	client := Nnrf_NFDiscovery.NewAPIClient(configuration)

	result, res, err := client.NFInstancesStoreApi.SearchNFInstances(context.TODO(), targetNfType, requestNfType, param)
	if res != nil && res.StatusCode == http.StatusTemporaryRedirect {
		err = fmt.Errorf("Temporary Redirect For Non NRF Consumer")
	}
	return result, err
}

func SearchUdmSdmInstance(ue *etaf_context.EtafUe, nrfUri string, targetNfType, requestNfType models.NfType,
	param *Nnrf_NFDiscovery.SearchNFInstancesParamOpts) error {

	resp, localErr := SendSearchNFInstances(nrfUri, targetNfType, requestNfType, param)
	if localErr != nil {
		return localErr
	}

	// select the first UDM_SDM, TODO: select base on other info
	var sdmUri string
	for _, nfProfile := range resp.NfInstances {
		ue.UdmId = nfProfile.NfInstanceId
		sdmUri = util.SearchNFServiceUri(nfProfile, models.ServiceName_NUDM_SDM, models.NfServiceStatus_REGISTERED)
		if sdmUri != "" {
			break
		}
	}
	ue.NudmSDMUri = sdmUri
	if ue.NudmSDMUri == "" {
		err := fmt.Errorf("ETAF can not select an UDM by NRF")
		logger.ConsumerLog.Errorf(err.Error())
		return err
	}
	return nil
}

func SearchNssfNSSelectionInstance(ue *etaf_context.EtafUe, nrfUri string, targetNfType, requestNfType models.NfType,
	param *Nnrf_NFDiscovery.SearchNFInstancesParamOpts) error {

	resp, localErr := SendSearchNFInstances(nrfUri, targetNfType, requestNfType, param)
	if localErr != nil {
		return localErr
	}

	// select the first NSSF, TODO: select base on other info
	var nssfUri string
	for _, nfProfile := range resp.NfInstances {
		ue.NssfId = nfProfile.NfInstanceId
		nssfUri = util.SearchNFServiceUri(nfProfile, models.ServiceName_NNSSF_NSSELECTION, models.NfServiceStatus_REGISTERED)
		if nssfUri != "" {
			break
		}
	}
	ue.NssfUri = nssfUri
	if ue.NssfUri == "" {
		return fmt.Errorf("ETAF can not select an NSSF by NRF")
	}
	return nil
}

func SearchAmfCommunicationInstance(ue *etaf_context.EtafUe, nrfUri string, targetNfType,
	requestNfType models.NfType, param *Nnrf_NFDiscovery.SearchNFInstancesParamOpts) (err error) {

	resp, localErr := SendSearchNFInstances(nrfUri, targetNfType, requestNfType, param)
	if localErr != nil {
		err = localErr
		return
	}

	var amfUri string
	for _, nfProfile := range resp.NfInstances {
		ue.AmfId = nfProfile.NfInstanceId
		amfUri = util.SearchNFServiceUri(nfProfile, models.ServiceName_NAMF_COMM, models.NfServiceStatus_REGISTERED)
		if amfUri != "" {
			break
		}
	}
	ue.AmfUri = amfUri
	if ue.AmfUri == "" {
		return fmt.Errorf("ETAF can not select an AMF by NRF")
	}
	return nil

}

func SearchAvailableAMFs(nrfUri string, serviceName models.ServiceName) (
	amfInfos []etaf_context.AMFStatusSubscriptionData) {
	localVarOptionals := Nnrf_NFDiscovery.SearchNFInstancesParamOpts{}

	result, err := SendSearchNFInstances(nrfUri, models.NfType_AMF, models.NfType_ETAF, &localVarOptionals)
	if err != nil {
		logger.ConsumerLog.Errorf(err.Error())
		return
	}

	for _, profile := range result.NfInstances {
		uri := util.SearchNFServiceUri(profile, serviceName, models.NfServiceStatus_REGISTERED)
		if uri != "" {
			item := etaf_context.AMFStatusSubscriptionData{
				AmfUri:    uri,
				GuamiList: *profile.AmfInfo.GuamiList,
			}
			amfInfos = append(amfInfos, item)
		}
	}
	return
}

