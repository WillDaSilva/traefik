package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
)

type WebProvider struct {
	Address           string
	CertFile, KeyFile string
}

type Page struct {
	Configurations configs
}

func (provider *WebProvider) Provide(configurationChan chan<- configMessage) {
	systemRouter := mux.NewRouter()
	systemRouter.Methods("GET").Path("/").Handler(http.HandlerFunc(GetHTMLConfigHandler))
	systemRouter.Methods("GET").Path("/health").Handler(http.HandlerFunc(GetHealthHandler))
	systemRouter.Methods("GET").Path("/api").Handler(http.HandlerFunc(GetConfigHandler))
	systemRouter.Methods("GET").Path("/api/{provider}").Handler(http.HandlerFunc(GetConfigHandler))
	systemRouter.Methods("PUT").Path("/api/{provider}").Handler(http.HandlerFunc(
		func(rw http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			if vars["provider"] != "web" {
				rw.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(rw, "Only 'web' provider can be updated through the REST API")
				return
			}

			configuration := new(Configuration)
			b, _ := ioutil.ReadAll(r.Body)
			err := json.Unmarshal(b, configuration)
			if err == nil {
				configurationChan <- configMessage{"web", configuration}
				GetConfigHandler(rw, r)
			} else {
				log.Errorf("Error parsing configuration %+v", err)
				http.Error(rw, fmt.Sprintf("%+v", err), http.StatusBadRequest)
			}
		}))
	systemRouter.Methods("GET").Path("/api/{provider}/backends").Handler(http.HandlerFunc(GetBackendsHandler))
	systemRouter.Methods("GET").Path("/api/{provider}/backends/{backend}").Handler(http.HandlerFunc(GetBackendHandler))
	systemRouter.Methods("GET").Path("/api/{provider}/backends/{backend}/servers").Handler(http.HandlerFunc(GetServersHandler))
	systemRouter.Methods("GET").Path("/api/{provider}/backends/{backend}/servers/{server}").Handler(http.HandlerFunc(GetServerHandler))
	systemRouter.Methods("GET").Path("/api/{provider}/frontends").Handler(http.HandlerFunc(GetFrontendsHandler))
	systemRouter.Methods("GET").Path("/api/{provider}/frontends/{frontend}").Handler(http.HandlerFunc(GetFrontendHandler))
	systemRouter.Methods("GET").PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "static"})))

	go func() {
		if len(provider.CertFile) > 0 && len(provider.KeyFile) > 0 {
			err := http.ListenAndServeTLS(provider.Address, provider.CertFile, provider.KeyFile, systemRouter)
			if err != nil {
				log.Fatal("Error creating server: ", err)
			}
		} else {
			err := http.ListenAndServe(provider.Address, systemRouter)
			if err != nil {
				log.Fatal("Error creating server: ", err)
			}
		}
	}()
}

func GetConfigHandler(rw http.ResponseWriter, r *http.Request) {
	templatesRenderer.JSON(rw, http.StatusOK, currentConfigurations)
}

func GetHTMLConfigHandler(response http.ResponseWriter, request *http.Request) {
	templatesRenderer.HTML(response, http.StatusOK, "configuration", Page{Configurations: currentConfigurations})
}

func GetHealthHandler(rw http.ResponseWriter, r *http.Request) {
	templatesRenderer.JSON(rw, http.StatusOK, metrics.Data())
}

func GetBackendsHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templatesRenderer.JSON(rw, http.StatusOK, currentConfigurations[vars["provider"]].Backends)
}

func GetBackendHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["backend"]
	if backend, ok := currentConfigurations[vars["provider"]].Backends[id]; ok {
		templatesRenderer.JSON(rw, http.StatusOK, backend)
	} else {
		http.NotFound(rw, r)
	}
}

func GetFrontendsHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templatesRenderer.JSON(rw, http.StatusOK, currentConfigurations[vars["provider"]].Frontends)
}

func GetFrontendHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["frontend"]
	if frontend, ok := currentConfigurations[vars["provider"]].Frontends[id]; ok {
		templatesRenderer.JSON(rw, http.StatusOK, frontend)
	} else {
		http.NotFound(rw, r)
	}
}

func GetServersHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backend := vars["backend"]
	if backend, ok := currentConfigurations[vars["provider"]].Backends[backend]; ok {
		templatesRenderer.JSON(rw, http.StatusOK, backend.Servers)
	} else {
		http.NotFound(rw, r)
	}
}

func GetServerHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backend := vars["backend"]
	server := vars["server"]
	if backend, ok := currentConfigurations[vars["provider"]].Backends[backend]; ok {
		if server, ok := backend.Servers[server]; ok {
			templatesRenderer.JSON(rw, http.StatusOK, server)
		} else {
			http.NotFound(rw, r)
		}
	} else {
		http.NotFound(rw, r)
	}
}
