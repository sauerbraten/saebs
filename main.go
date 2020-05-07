package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/lang/en"
	"github.com/blevesearch/bleve/mapping"
	"github.com/caltechlibrary/bibtex"
)

func main() {
	s := NewServer()
	fmt.Println(http.ListenAndServe(":8080", s.router))
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
		if len(ref.Keys) < 1 {
			return nil, fmt.Errorf("reference has no keys:\n%s", ref)
		}
		err = index.Index(ref.Keys[0], ref)
		if err != nil {
			return nil, fmt.Errorf("indexing %s: %w", ref.Keys[0], err)
		}
		log.Println("indexed", ref.Keys[0])
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
