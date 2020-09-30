package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/caltechlibrary/bibtex"
	"github.com/go-chi/chi"
)

type Server struct {
	index  bleve.Index
	mu     sync.RWMutex
	router chi.Router
}

func NewServer() *Server {
	// try to restore mapping from fs
	i, err := findLatestIndex()
	if err != nil {
		fmt.Println(err)
		fmt.Println("falling back to in-memory dummy index")
		// create dummy index to prevent nil pointer
		i, err = bleve.NewMemOnly(bleve.NewIndexMapping())
		if err != nil {
			panic(err)
		}
	}
	s := &Server{
		index: i,
	}
	s.setupRouter()
	return s
}

func (s *Server) setupRouter() {
	r := chi.NewMux()
	r.Get("/", s.landingPage)
	r.Post("/recreateIndex", s.recreateIndex)
	r.Get("/stats", s.stats)
	r.Get("/search", s.search)
	r.Handle("/{:[a-z]+\\.css}", http.FileServer(http.Dir("css")))
	s.router = r
}

func (s *Server) swapIndex(index bleve.Index) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.index.Close()
	if err != nil {
		return fmt.Errorf("closing old index: %w", err)
	}
	s.index = index
	// TODO: remove old index?
	return nil
}

func (s *Server) landingPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.
		New("base.tmpl"). // must be the base template (entry point) so templates are associated correctly by ParseFiles()
		Funcs(template.FuncMap{"titlecase": strings.Title}).
		Option("missingkey=error").
		ParseFiles("templates/base.tmpl", "templates/search_form.tmpl", "templates/upload_form.tmpl", "templates/front.tmpl")
	if err != nil {
		log.Fatalln(err)
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		log.Println(err)
	}
}

func (s *Server) recreateIndex(w http.ResponseWriter, r *http.Request) {
	// read uploaded file to memory
	bibtexFile, _, err := r.FormFile("bibtex")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "error:", err)
		return
	}

	bibtexContents, err := ioutil.ReadAll(bibtexFile)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "error:", err)
		return
	}

	// recreate index
	index, err := buildIndex(bibtexContents)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "error:", err)
		return
	}

	s.swapIndex(index)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	docCount, _ := s.index.DocCount()
	w.Header().Add("content-type", "application/json")
	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"name":      s.index.Name(),
			"stats":     s.index.Stats(),
			"doc_count": docCount,
		},
	)
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	log.Println(r.FormValue("q"))

	// parse query from request
	queryString := strings.TrimSpace(r.FormValue("q"))
	if queryString == "" {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	queryExpression := parseQuery(queryString)

	// build boolean query of query string queries
	var buildQuery func(expr *Expr) query.Query
	buildQuery = func(expr *Expr) query.Query {
		if expr.lit != "" {
			return query.NewBooleanQuery([]query.Query{query.NewQueryStringQuery(expr.lit)}, nil, nil)
		}

		queries := []query.Query{buildQuery(expr.left), buildQuery(expr.right)}
		switch expr.op {
		case "AND":
			return query.NewConjunctionQuery(queries)
		case "OR":
			return query.NewDisjunctionQuery(queries)
		default:
			fmt.Println("unhandled query operator", expr.op)
			return nil
		}
	}

	query := buildQuery(queryExpression)
	if query == nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "could not parse query")
		return
	}

	search := bleve.NewSearchRequestOptions(query, 100_000, 0, false)
	search.Fields = []string{"*"}
	searchResults, err := s.index.Search(search)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
		return
	}

	if r.FormValue("format") == "bibfile" {
		w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.bib"`, r.FormValue("q")))
		w.WriteHeader(http.StatusOK)
		for _, hit := range searchResults.Hits {
			tags := map[string]string{}

			// collect tags
			for k, v := range hit.Fields {
				if !strings.HasPrefix(k, "tags.") {
					continue
				}
				// cut 'tags.' from key
				k = k[len("tags."):]
				// set cleaned up value
				tags[k] = strings.Trim(v.(string), "{}")
			}

			e := &bibtex.Element{
				ID:   hit.Fields["keys"].(string),
				Type: hit.Fields["type"].(string),
				Tags: tags,
			}

			w.Write([]byte(e.String()))
		}
		return
	}

	lookup := NewLookup(queryExpression, searchResults)

	tmpl, err := template.
		New("base.tmpl"). // must be the base template (entry point) so templates are associated correctly by ParseFiles()
		Funcs(template.FuncMap{"titlecase": strings.Title}).
		Option("missingkey=error").
		ParseFiles("templates/base.tmpl", "templates/search_form.tmpl", "templates/upload_form.tmpl", "templates/search_results.tmpl")
	if err != nil {
		log.Fatalln(err)
	}

	err = tmpl.Execute(w, lookup)
	if err != nil {
		log.Println(err)
	}
}

type Result struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Authors     string            `json:"authors"`
	Year        string            `json:"year"`
	Abstract    string            `json:"abstract"`
	Keywords    string            `json:"keywords"`
	OtherFields map[string]string `json:"other_fields"`
}

func NewResult(id string, fields map[string]interface{}) Result {
	tags := map[string]string{}
	for k, _v := range fields {
		if strings.HasPrefix(k, "tags.") {
			// cut 'tags.' from key
			k = k[len("tags."):]
		}
		v, ok := _v.(string)
		if !ok {
			panic(fmt.Errorf("value of field %s in result %s is not of type string", k, id))
		}
		tags[k] = strings.Trim(v, "{}")
		// TODO more unescaping
	}

	r := Result{
		ID:       id,
		Title:    tags["title"],
		Authors:  tags["author"],
		Year:     tags["year"],
		Abstract: tags["abstract"],
		Keywords: strings.Join(strings.Split(tags["keywords"], ","), ", "),
	}

	// delete tags we have explizit fields for
	delete(tags, "title")
	delete(tags, "author")
	delete(tags, "year")
	delete(tags, "abstract")
	delete(tags, "keywords")

	// delete tags we never need in the frontend
	delete(tags, "address")
	delete(tags, "uid")
	delete(tags, "UID")
	delete(tags, "doi")
	delete(tags, "id")
	delete(tags, "keys")
	delete(tags, "issn")
	delete(tags, "editor")
	delete(tags, "type")
	delete(tags, "url")

	r.OtherFields = tags

	return r
}

type FinishedLookup struct {
	Query   string   `json:"query"`
	Results []Result `json:"results"`
}

func NewLookup(query *Expr, searchResults *bleve.SearchResult) FinishedLookup {
	lookup := FinishedLookup{
		Query:   query.String(),
		Results: make([]Result, searchResults.Hits.Len()),
	}

	for i, hit := range searchResults.Hits {
		lookup.Results[i] = NewResult(hit.ID, hit.Fields)
	}

	return lookup
}
