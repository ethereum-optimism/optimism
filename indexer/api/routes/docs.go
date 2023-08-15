package routes

import (
	"net/http"

	"github.com/go-chi/docgen"
)

func (h Routes) DocsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	docs := docgen.MarkdownRoutesDoc(h.Router, docgen.MarkdownOpts{
		ProjectPath: "github.com/ethereum-optimism/optimism/indexer",
		// Intro text included at the top of the generated markdown file.
		Intro: "Generated documentation for Optimism indexer",
	})
	_, err := w.Write([]byte(docs))
	if err != nil {
		h.Logger.Error("error writing docs", "err", err)
		http.Error(w, "Internal server error fetching docs", http.StatusInternalServerError)
	}
}
