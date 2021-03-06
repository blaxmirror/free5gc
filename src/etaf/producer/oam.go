package producer

import (
	"free5gc/lib/http_wrapper"
	"free5gc/lib/openapi/models"
	"free5gc/src/etaf/context"
	"free5gc/src/etaf/logger"
	"net/http"
	"strconv"
)

type PduSession struct {
	PduSessionId string
	SmContextRef string
	Sst          string
	Sd           string
	Dnn          string
}

type UEContext struct {
	AccessType models.AccessType
	Supi       string
	Guti       string
	/* Tai */
	Mcc string
	Mnc string
	Tac string
	/* PDU sessions */
	PduSessions []PduSession
	/*Connection state */
	CmState models.CmState
}

type UEContexts []UEContext

func HandleOAMRegisteredUEContext(request *http_wrapper.Request) *http_wrapper.Response {
	logger.ProducerLog.Infof("[OAM] Handle Registered UE Context")

	supi := request.Params["supi"]

	ueContexts, problemDetails := OAMRegisteredUEContextProcedure(supi)
	if problemDetails != nil {
		return http_wrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	} else {
		return http_wrapper.NewResponse(http.StatusOK, nil, ueContexts)
	}
}

func OAMRegisteredUEContextProcedure(supi string) (UEContexts, *models.ProblemDetails) {
	var ueContexts UEContexts
	etafSelf := context.ETAF_Self()

	if supi != "" {
		if ue, ok := etafSelf.EtafUeFindBySupi(supi); ok {
			ueContext := buildUEContext(ue, models.AccessType__3_GPP_ACCESS)
			if ueContext != nil {
				ueContexts = append(ueContexts, *ueContext)
			}
			ueContext = buildUEContext(ue, models.AccessType_NON_3_GPP_ACCESS)
			if ueContext != nil {
				ueContexts = append(ueContexts, *ueContext)
			}
		} else {
			problemDetails := &models.ProblemDetails{
				Status: http.StatusNotFound,
				Cause:  "CONTEXT_NOT_FOUND",
			}
			return nil, problemDetails
		}
	} else {
		etafSelf.UePool.Range(func(key, value interface{}) bool {
			ue := value.(*context.EtafUe)
			ueContext := buildUEContext(ue, models.AccessType__3_GPP_ACCESS)
			if ueContext != nil {
				ueContexts = append(ueContexts, *ueContext)
			}
			ueContext = buildUEContext(ue, models.AccessType_NON_3_GPP_ACCESS)
			if ueContext != nil {
				ueContexts = append(ueContexts, *ueContext)
			}
			return true
		})
	}

	return ueContexts, nil
}
func buildUEContext(ue *context.EtafUe, accessType models.AccessType) *UEContext {
	if ue.State[accessType].Is(context.Registered) {
		ueContext := &UEContext{
			AccessType: models.AccessType__3_GPP_ACCESS,
			Supi:       ue.Supi,
			Guti:       ue.Guti,
			Mcc:        ue.Tai.PlmnId.Mcc,
			Mnc:        ue.Tai.PlmnId.Mnc,
			Tac:        ue.Tai.Tac,
		}

		for _, smContext := range ue.SmContextList {
			pduSessionContext := smContext.PduSessionContext
			if pduSessionContext != nil {
				if pduSessionContext.AccessType == accessType {
					pduSession := PduSession{
						PduSessionId: strconv.Itoa(int(pduSessionContext.PduSessionId)),
						SmContextRef: pduSessionContext.SmContextRef,
						Sst:          strconv.Itoa(int(pduSessionContext.SNssai.Sst)),
						Sd:           pduSessionContext.SNssai.Sd,
						Dnn:          pduSessionContext.Dnn,
					}
					ueContext.PduSessions = append(ueContext.PduSessions, pduSession)
				}
			}
		}

		if ue.CmConnect(accessType) {
			ueContext.CmState = models.CmState_CONNECTED
		} else {
			ueContext.CmState = models.CmState_IDLE
		}
		return ueContext
	}
	return nil
}
