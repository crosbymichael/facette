package server

import (
	"encoding/json"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/facette/facette/pkg/library"
	"github.com/facette/facette/pkg/logger"
	"github.com/facette/facette/thirdparty/github.com/fatih/set"
)

func (server *Server) serveError(writer http.ResponseWriter, status int) {
	err := server.execTemplate(
		writer,
		status,
		struct {
			URLPrefix string
			ReadOnly  bool
			Status    int
		}{
			URLPrefix: server.Config.URLPrefix,
			ReadOnly:  server.Config.ReadOnly,
			Status:    status,
		},
		path.Join(server.Config.BaseDir, "template", "layout.html"),
		path.Join(server.Config.BaseDir, "template", "error.html"),
	)

	if err != nil {
		logger.Log(logger.LevelError, "server", "%s", err)
		server.serveResponse(writer, nil, status)
	}
}

func (server *Server) serveReload(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" && request.Method != "HEAD" {
		server.serveResponse(writer, serverResponse{mesgMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	} else if server.Config.ReadOnly {
		server.serveResponse(writer, serverResponse{mesgReadOnlyMode}, http.StatusForbidden)
		return
	}

	// Reload resources without reloading configuration
	server.Reload(false)

	server.serveResponse(writer, nil, http.StatusOK)
}

func (server *Server) serveResponse(writer http.ResponseWriter, data interface{}, status int) {
	var (
		err    error
		output []byte
	)

	if data != nil {
		output, err = json.Marshal(data)
		if err != nil {
			server.serveResponse(writer, nil, http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	writer.WriteHeader(status)

	if len(output) > 0 {
		writer.Write(output)
		writer.Write([]byte("\n"))
	}
}

func (server *Server) serveStatic(writer http.ResponseWriter, request *http.Request) {
	mimeType := mime.TypeByExtension(filepath.Ext(request.URL.Path))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	writer.Header().Set("Content-Type", mimeType)

	// Handle static files
	http.ServeFile(writer, request, path.Join(server.Config.BaseDir, request.URL.Path))
}

func (server *Server) serveStats(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" && request.Method != "HEAD" {
		server.serveResponse(writer, serverResponse{mesgMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	server.serveResponse(writer, server.getStats(writer, request), http.StatusOK)
}

func (server *Server) serveWait(writer http.ResponseWriter, request *http.Request) {
	err := server.execTemplate(
		writer,
		http.StatusServiceUnavailable,
		struct {
			URLPrefix string
			ReadOnly  bool
		}{
			URLPrefix: server.Config.URLPrefix,
			ReadOnly:  server.Config.ReadOnly,
		},
		path.Join(server.Config.BaseDir, "template", "layout.html"),
		path.Join(server.Config.BaseDir, "template", "wait.html"),
	)

	if err != nil {
		if os.IsNotExist(err) {
			server.serveError(writer, http.StatusNotFound)
		} else {
			logger.Log(logger.LevelError, "server", "%s", err)
			server.serveError(writer, http.StatusInternalServerError)
		}
	}
}

func (server *Server) getStats(writer http.ResponseWriter, request *http.Request) *statsResponse {
	sourceSet := set.New(set.ThreadSafe)
	metricSet := set.New(set.ThreadSafe)

	for _, origin := range server.Catalog.Origins {
		for key, source := range origin.Sources {
			sourceSet.Add(key)

			for key := range source.Metrics {
				metricSet.Add(key)
			}
		}
	}

	sourceGroupsCount := 0
	metricGroupsCount := 0

	for _, group := range server.Library.Groups {
		if group.Type == library.LibraryItemSourceGroup {
			sourceGroupsCount++
		} else {
			metricGroupsCount++
		}
	}

	return &statsResponse{
		Origins:      len(server.Catalog.Origins),
		Sources:      sourceSet.Size(),
		Metrics:      metricSet.Size(),
		Graphs:       len(server.Library.Graphs),
		Collections:  len(server.Library.Collections),
		SourceGroups: sourceGroupsCount,
		MetricGroups: metricGroupsCount,
	}
}
