//
// Copyright (C) 2020 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/go-mod-bootstrap/v3/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v3/bootstrap/handlers"
	"github.com/edgexfoundry/go-mod-bootstrap/v3/bootstrap/interfaces"
	"github.com/edgexfoundry/go-mod-bootstrap/v3/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/common"
	commonDTO "github.com/edgexfoundry/go-mod-core-contracts/v3/dtos/common"
	"github.com/gorilla/mux"
	"net/http"
)

// CommonController controller for common REST APIs
type CommonController struct {
	serviceName  string
	router       *mux.Router
	version      version
	customConfig interfaces.UpdatableConfig
	lc           logger.LoggingClient
	dic          *di.Container
}

type version struct {
	serviceVersion string
	sdkVersion     string
}

func NewCommonController(r *mux.Router, dic *di.Container, serviceName string) *CommonController {
	lc := container.LoggingClientFrom(dic.Get)
	//version := version{
	//	serviceVersion: ,
	//	sdkVersion: ,
	//}
	return &CommonController{
		serviceName: serviceName,
		router:      r,
		lc:          lc,
		dic:         dic,
	}
}

// SetVersion sets the service's version and skd version if used.
func (c *CommonController) SetVersion(serviceVersion, sdkVersion string) {
	c.version.serviceVersion = serviceVersion
	c.version.sdkVersion = sdkVersion
}

// SetCustomConfigInfo sets the custom configuration, which is used to include the service's custom config in the /config endpoint response.
func (c *CommonController) SetCustomConfigInfo(customConfig interfaces.UpdatableConfig) {
	c.customConfig = customConfig
}

func LoadCommonRoutes(r *mux.Router, dic *di.Container, serviceName string) {
	c := NewCommonController(r, dic, serviceName)
	secretProvider := container.SecretProviderExtFrom(dic.Get)
	authenticationHook := handlers.AutoConfigAuthenticationFunc(secretProvider, c.lc)

	r.HandleFunc(common.ApiPingRoute, c.Ping).Methods(http.MethodGet) // Health check is always unauthenticated
	r.HandleFunc(common.ApiVersionRoute, authenticationHook(c.Version)).Methods(http.MethodGet)
	//r.HandleFunc(common.ApiConfigRoute, authenticationHook(c.Config)).Methods(http.MethodGet)
}

// Ping handles the request to /ping endpoint. Is used to test if the service is working
// It returns a response as specified by the API swagger in the openapi directory
func (c *CommonController) Ping(writer http.ResponseWriter, request *http.Request) {
	response := commonDTO.NewPingResponse(c.serviceName)
	c.sendResponse(writer, request, common.ApiPingRoute, response, http.StatusOK)
}

// Version handles the request to /version endpoint. Is used to request the service's versions
// It returns a response as specified by the API swagger in the openapi directory
func (c *CommonController) Version(writer http.ResponseWriter, request *http.Request) {
	var response interface{}
	if c.version.sdkVersion != "" {
		response = commonDTO.NewVersionSdkResponse(c.version.serviceVersion, c.version.sdkVersion, c.serviceName)
	} else {
		response = commonDTO.NewVersionResponse(c.version.serviceVersion, c.serviceName)
	}
	c.sendResponse(writer, request, common.ApiVersionRoute, response, http.StatusOK)
}

// Config handles the request to /config endpoint. Is used to request the service's configuration
// It returns a response as specified by the V2 API swagger in openapi/common
//func (c *CommonController) Config(writer http.ResponseWriter, request *http.Request) {
//	var fullConfig interface{}
//	configuration := container.ConfigurationFrom(c.dic.Get)
//
//	if c.customConfig == nil {
//		// case of no custom configs
//		fullConfig = *configuration
//	} else {
//		// create a struct combining the common configuration and custom configuration sections
//		fullConfig = struct {
//			config.ConfigurationStruct
//			CustomConfiguration interfaces.UpdatableConfig
//		}{
//			*configuration,
//			c.customConfig,
//		}
//	}
//
//	response := commonDTO.NewConfigResponse(fullConfig, c.serviceName)
//	c.sendResponse(writer, request, common.ApiVersionRoute, response, http.StatusOK)
//}

// sendResponse puts together the response packet for the V2 API
func (c *CommonController) sendResponse(
	writer http.ResponseWriter,
	request *http.Request,
	api string,
	response interface{},
	statusCode int) {

	correlationID := request.Header.Get(common.CorrelationHeader)

	writer.Header().Set(common.CorrelationHeader, correlationID)
	writer.Header().Set(common.ContentType, common.ContentTypeJSON)
	writer.WriteHeader(statusCode)

	if response != nil {
		data, err := json.Marshal(response)
		if err != nil {
			c.lc.Error(fmt.Sprintf("Unable to marshal %s response", api), "error", err.Error(), common.CorrelationHeader, correlationID)
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = writer.Write(data)
		if err != nil {
			c.lc.Error(fmt.Sprintf("Unable to write %s response", api), "error", err.Error(), common.CorrelationHeader, correlationID)
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
