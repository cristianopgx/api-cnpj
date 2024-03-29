package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cuducos/go-cnpj"
	"github.com/cuducos/minha-receita/monitor"
	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/logWriter"
	"github.com/newrelic/go-agent/v3/newrelic"
)

const cacheMaxAge = time.Hour * 24

var cacheControl = fmt.Sprintf("max-age=%d", int(cacheMaxAge.Seconds()))

type database interface {
	GetCompany(string) (string, error)
	MetaRead(string) (string, error)
	Search(req searchRequest) ([]interface{}, error)
}

type api struct {
	db          database
	host        string
	errorLogger logWriter.LogWriter
}

type errorMessage struct {
	Message string `json:"message"`
}

type searchRequest struct {
	Page    int `json:"page"`
	Results int `json:"results"`
	// Adicione aqui os campos de filtro desejados
}

func (app *api) messageResponse(w http.ResponseWriter, s int, m string) {
	if m == "" {
		w.WriteHeader(s)
		if s == http.StatusInternalServerError {
			app.errorLogger.Write([]byte("Internal server error without error message"))
		}
		return
	}

	b, err := json.Marshal(errorMessage{m})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.errorLogger.Write([]byte(fmt.Sprintf("Could not wrap message in JSON: %s", m)))
		return
	}

	w.WriteHeader(s)
	w.Header().Set("Content-type", "application/json")
	w.Write(b)

	if s == http.StatusInternalServerError {
		app.errorLogger.Write(b)
	}
}

func (app *api) companyHandler(w http.ResponseWriter, r *http.Request) {
	// Implementação do handler companyHandler
}

func (app *api) updatedHandler(w http.ResponseWriter, r *http.Request) {
	// Implementação do handler updatedHandler
}

func (app *api) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Implementação do handler healthHandler
}

func (app *api) searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.messageResponse(w, http.StatusMethodNotAllowed, "Este endpoint aceita apenas o método POST.")
		return
	}

	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.messageResponse(w, http.StatusBadRequest, "Erro ao decodificar a solicitação JSON.")
		return
	}

	// Defina um limite padrão para o número de resultados por página
	if req.Results <= 0 {
		req.Results = 100
	}

	// Defina a página padrão como 1
	if req.Page <= 0 {
		req.Page = 1
	}

	// Calcule o offset com base na página solicitada
	offset := (req.Page - 1) * req.Results

	// Realize a consulta no banco de dados usando os critérios de pesquisa
	// e a configuração de paginação
	results, err := app.db.Search(req)
	if err != nil {
		app.messageResponse(w, http.StatusInternalServerError, "Erro ao realizar a pesquisa.")
		return
	}

	// Simule uma resposta com os resultados
	type searchResponse struct {
		Results []interface{} `json:"results"`
		Page    int           `json:"page"`
		Total   int           `json:"total"`
	}

	resp := searchResponse{
		Results: results,
		Page:    req.Page,
		Total:   len(results),
	}

	// Serialize a resposta em JSON
	w.Header().Set("Content-type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		app.messageResponse(w, http.StatusInternalServerError, "Erro ao serializar a resposta JSON.")
		return
	}
}

func (app *api) allowedHostWrapper(h func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	// Implementação do allowedHostWrapper
}

func Serve(db database, p string, nr *newrelic.Application) {
	if !strings.HasPrefix(p, ":") {
		p = ":" + p
	}
	app := api{
		db:          db,
		host:        os.Getenv("ALLOWED_HOST"),
		errorLogger: logWriter.New(os.Stderr, nr),
	}
	for _, r := range []struct {
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"/", app.companyHandler},
		{"/updated", app.updatedHandler},
		{"/healthz", app.healthHandler},
		{"/search", app.searchHandler}, // Adicione o novo endpoint de pesquisa
	} {
		http.HandleFunc(monitor.NewRelicHandle(nr, r.path, app.allowedHostWrapper(r.handler)))
	}
	log.Output(1, fmt.Sprintf("Serving at http://0.0.0.0%s", p))
	log.Fatal(http.ListenAndServe(p, nil))
}
