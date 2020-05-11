package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/lang/en"
	"github.com/blevesearch/bleve/mapping"
	"github.com/caltechlibrary/bibtex"

	"saebs/uuid"
)

func main() {
	s := NewServer()
	fmt.Println(http.ListenAndServe(":8084", s.router))
}

func buildIndex(bibtexFile []byte) (bleve.Index, error) {
	// create new index
	mapping := buildIndexMapping()
	indexName, err := ioutil.TempDir(".", strconv.FormatInt(time.Now().Unix(), 10)+"_*")
	if err != nil {
		return nil, fmt.Errorf("creating index name: %w", err)
	}
	index, err := bleve.New(indexName, mapping)
	if err != nil {
		return nil, fmt.Errorf("creating index: %w", err)
	}

	// parse file contents into bibtex references
	references, err := bibtex.Parse(bibtexFile)
	if err != nil {
		return nil, fmt.Errorf("parsing bibtex data: %w", err)
	}

	fmt.Println("parsed", len(references), "references from bibtex file")

	// index in bleve
	for _, ref := range references {
		err = index.Index(uuid.NewV4().String(), ref)
		if err != nil {
			return nil, fmt.Errorf("indexing %s: %w", ref.Keys[0], err)
		}
	}

	return index, nil
}

func buildIndexMapping() mapping.IndexMapping {
	// a generic reusable mapping for english text
	englishTextFieldMapping := bleve.NewTextFieldMapping()
	englishTextFieldMapping.Analyzer = en.AnalyzerName

	bibtexElementMapping := bleve.NewDocumentMapping()

	bibtexElementMapping.AddFieldMappingsAt("Tags.abstract", englishTextFieldMapping)
	bibtexElementMapping.AddFieldMappingsAt("Tags.keywords", englishTextFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultMapping = bibtexElementMapping
	indexMapping.DefaultAnalyzer = "en"

	return indexMapping
}
