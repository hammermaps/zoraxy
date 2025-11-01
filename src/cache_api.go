package main

import (
	"encoding/json"
	"net/http"

	"imuslab.com/zoraxy/mod/dynamicproxy"
	"imuslab.com/zoraxy/mod/utils"
)

/*
	cache_api.go

	This file contains API handlers for per-host cache configuration
*/

// HandleGetHostCacheSettings retrieves cache settings for a specific host
func HandleGetHostCacheSettings(w http.ResponseWriter, r *http.Request) {
	hostname, err := utils.GetPara(r, "hostname")
	if err != nil {
		utils.SendErrorResponse(w, "hostname is required")
		return
	}

	// Load the proxy endpoint
	ep, err := dynamicProxyRouter.LoadProxy(hostname)
	if err != nil {
		utils.SendErrorResponse(w, "Proxy endpoint not found")
		return
	}

	// Return cache settings
	if ep.CacheSettings == nil {
		// Return default settings indicating use of global config
		defaultSettings := &dynamicproxy.HostCacheSettings{
			UseGlobal: true,
		}
		js, _ := json.Marshal(defaultSettings)
		utils.SendJSONResponse(w, string(js))
	} else {
		js, _ := json.Marshal(ep.CacheSettings)
		utils.SendJSONResponse(w, string(js))
	}
}

// HandleSetHostCacheSettings updates cache settings for a specific host
func HandleSetHostCacheSettings(w http.ResponseWriter, r *http.Request) {
	hostname, err := utils.GetPara(r, "hostname")
	if err != nil {
		utils.SendErrorResponse(w, "hostname is required")
		return
	}

	// Load the proxy endpoint
	ep, err := dynamicProxyRouter.LoadProxy(hostname)
	if err != nil {
		utils.SendErrorResponse(w, "Proxy endpoint not found")
		return
	}

	// Parse the cache settings from request body
	var cacheSettings dynamicproxy.HostCacheSettings
	err = json.NewDecoder(r.Body).Decode(&cacheSettings)
	if err != nil {
		utils.SendErrorResponse(w, "Invalid cache settings format")
		return
	}

	// Update the proxy endpoint with new cache settings
	ep.CacheSettings = &cacheSettings
	ep.UpdateToRuntime()

	// Save the configuration
	err = SaveReverseProxyConfig(ep)
	if err != nil {
		utils.SendErrorResponse(w, "Failed to save cache settings: "+err.Error())
		return
	}

	utils.SendOK(w)
}
