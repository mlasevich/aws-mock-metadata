package main

import (
	"fmt"

	"github.com/gorilla/mux"
)

// NewServer creates a new http server (starting handled separately to allow
// test suites to reuse)
func (app *App) NewServer() *mux.Router {
	r := mux.NewRouter()
	r.Handle("", appHandler(app.rootHandler))
	r.Handle("/", appHandler(app.rootHandler))

	for _, v := range app.apiVersionPrefixes() {
		app.versionSubRouter(r.PathPrefix(fmt.Sprintf("/%s", v)).Subrouter(), v)
	}

	r.Handle("/{path:.*}", appHandler(app.notFoundHandler))

	return r
}

// Provides the versioned (normally 1.0, YYYY-MM-DD or latest) prefix routes
// TODO: conditional out the namespaces that don't exist on selected API versions
func (app *App) versionSubRouter(router *mux.Router, version string) {
	//router.Handle("", appHandler(app.trailingSlashRedirect))
	app.addEndpoint(router, "", appHandler(app.secondLevelHandler))

	app.addDirectory(router, "/dynamic", version, appHandler(app.dynamicHandler),
		func(router *mux.Router, version string) {
			app.addDirectory(router, "/instance-identity", version,
				appHandler(app.instanceIdentityHandler),
				func(router *mux.Router, version string) {
					app.addEndpoint(router, "/document",
						appHandler(app.instanceIdentityDocumentHandler))
					app.addEndpoint(router, "/pkcs7",
						appHandler(app.instanceIdentityPkcs7Handler))
					app.addEndpoint(router, "/signature",
						appHandler(app.instanceIdentitySignatureHandler))
				},
			)
		},
	)

	app.addDirectory(router, "/api", version, nil,
		func(router *mux.Router, version string) {

			// For IMDSv2, https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/configuring-instance-metadata-service.html
			// DO we need this?
			app.addEndpoint(router, "", appHandler(app.notFoundHandler))

			router.Handle("/token", appHandler(app.apiTokenHandler)).Methods("PUT")
			// TODO: return 405 for everything but PUT
			/*
				HTTP/1.1 405 Not Allowed
				Allow: OPTIONS, PUT
				Content-Length: 0
				Date: Tue, 07 Apr 2020 03:56:56 GMT
				Server: EC2ws
				Connection: close
				Content-Type: text/plain
			*/
			router.Handle("/token", appHandler(app.apiTokenNotPutHandler)).Methods("GET", "POST", "DELETE")
		},
	)

	app.addDirectory(router, "/meta-data", version, appHandler(app.metaDataHandler),
		subRouterDef(app.addRoutesMetaData))

	// Add 404 everywhere
	err := router.Walk(func(route *mux.Route, subRouter *mux.Router, ancestors []*mux.Route) error {
		subRouter.Handle("/{path:.*}", appHandler(app.notFoundHandler))
		return nil
	})
	if err != nil {
		println("error setting up routes")
	}
}

// Wrapper for Router Definition Function
type subRouterDef func(*mux.Router, string)

// General Add Subrouter
func (app *App) addDirectory(parent *mux.Router, route string, version string,
	dirHandler appHandler, definition subRouterDef) *mux.Router {
	router := parent.PathPrefix(route).Subrouter()
	if dirHandler != nil {
		router.Handle("", appHandler(app.trailingSlashRedirect))
		router.Handle("/", dirHandler)
	}
	if definition != nil {
		definition(router, version)
	}
	return router
}

// Handles route same with or without a trailing slash
func (app *App) addEndpoint(router *mux.Router, route string, handler appHandler) {
	router.Handle(route, handler)
	router.Handle(route+"/", handler)
}

// Add '/meta-data' routes
func (app *App) addRoutesMetaData(router *mux.Router, version string) {
	app.addEndpoint(router, "/ami-id", appHandler(app.amiIdHandler))
	app.addEndpoint(router, "/ami-launch-index", appHandler(app.amiLaunchIndexHandler))
	app.addEndpoint(router, "/ami-manifest-path", appHandler(app.amiManifestPathHandler))
	// events/maintenance/history ?

	app.addDirectory(router, "/block-device-mapping", version, appHandler(app.blockDeviceMappingHandler),
		func(router *mux.Router, version string) {
			app.addEndpoint(router, "/ami", appHandler(app.blockDeviceMappingAmiHandler))
			app.addEndpoint(router, "/root", appHandler(app.blockDeviceMappingRootHandler))
		},
	)

	app.addEndpoint(router, "/hostname", appHandler(app.hostnameHandler))

	app.addDirectory(router, "/iam", version, appHandler(app.iamHandler),
		func(router *mux.Router, version string) {
			app.addEndpoint(router, "/info", appHandler(app.infoHandler))
			app.addDirectory(router, "/security-credentials", version, appHandler(app.securityCredentialsHandler),
				func(router *mux.Router, version string) {
					//TODO: join these into one
					if app.MockInstanceProfile {
						app.addEndpoint(router, "/"+app.RoleName, appHandler(app.mockRoleHandler))
					} else {
						app.addEndpoint(router, "/"+app.RoleName, appHandler(app.roleHandler))
					}
				},
			)
		},
	)

	// identity-credentials/ ?
	app.addEndpoint(router, "/instance-action", appHandler(app.instanceActionHandler))
	app.addEndpoint(router, "/instance-id", appHandler(app.instanceIDHandler))
	// instance-life-cycle
	app.addEndpoint(router, "/instance-type", appHandler(app.instanceTypeHandler))
	app.addEndpoint(router, "/local-hostname", appHandler(app.localHostnameHandler))
	app.addEndpoint(router, "/local-ipv4", appHandler(app.privateIpHandler))
	app.addEndpoint(router, "/mac", appHandler(app.macHandler))

	app.addDirectory(router, "/metrics", version, appHandler(app.metricsHandler),
		func(router *mux.Router, version string) {
			app.addEndpoint(router, "/vhostmd", appHandler(app.metricsVhostmdHandler))
		},
	)

	app.addDirectory(router, "/network", version, appHandler(app.networkHandler),
		func(router *mux.Router, version string) {
			app.addDirectory(router, "/interfaces", version, appHandler(app.networkInterfacesHandler),
				func(router *mux.Router, version string) {
					app.addDirectory(router, "/macs", version, appHandler(app.networkInterfacesMacsHandler),
						func(router *mux.Router, version string) {
							app.addDirectory(router, "/"+app.MacAddress, version, appHandler(app.networkInterfacesMacsAddrHandler),
								func(router *mux.Router, version string) {
									app.addEndpoint(router, "/device-number", appHandler(app.nimAddrDeviceNumberHandler))
									app.addEndpoint(router, "/interface-id", appHandler(app.nimAddrInterfaceIdHandler))
									// TODO: expand API coverage
									app.addEndpoint(router, "/vpc-id", appHandler(app.vpcHandler))
								},
							)
						},
					)
				},
			)
		},
	)

	app.addDirectory(router, "/placement", version, appHandler(app.placementHandler),
		func(router *mux.Router, version string) {
			app.addEndpoint(router, "/availability-zone", appHandler(app.availabilityZoneHandler))
			app.addEndpoint(router, "/region", appHandler(app.regionHandler))

		},
	)

	app.addEndpoint(router, "/profile", appHandler(app.profileHandler))
	app.addEndpoint(router, "/public-hostname", appHandler(app.hostnameHandler))
	//public-keys/
	//reservation-id
	//security-groups
	//services
}
