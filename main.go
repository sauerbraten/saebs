package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

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

const LatestIndexDiscoveryFile = "latest_index"

func findLatestIndex() (bleve.Index, error) {
	contents, err := ioutil.ReadFile(LatestIndexDiscoveryFile)
	if err != nil {
		return nil, fmt.Errorf("finding latest existing index: %w", err)
	}
	indexName := string(contents)
	return bleve.Open(indexName)
}

func buildIndex(bibtexFile []byte) (bleve.Index, error) {
	// create new index
	indexName, err := ioutil.TempDir(".", "index_")
	if err != nil {
		return nil, fmt.Errorf("creating index name: %w", err)
	}
	err = ioutil.WriteFile(LatestIndexDiscoveryFile, []byte(indexName), 0644)
	if err != nil {
		return nil, fmt.Errorf("writing new index name to discovery file: %w", err)
	}
	index, err := bleve.New(indexName, buildIndexMapping())
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

	bibtexElementMapping := bleve.NewDocumentStaticMapping()

	bibtexElementMapping.AddFieldMappingsAt("Tags.title", englishTextFieldMapping)
	bibtexElementMapping.AddFieldMappingsAt("Tags.abstract", englishTextFieldMapping)
	bibtexElementMapping.AddFieldMappingsAt("Tags.keywords", englishTextFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultMapping = bibtexElementMapping
	indexMapping.DefaultAnalyzer = "en"

	return indexMapping
}
