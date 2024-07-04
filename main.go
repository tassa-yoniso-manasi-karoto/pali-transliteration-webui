package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	palitrans "github.com/siongui/pali-transliteration"
	pli "github.com/tassa-yoniso-manasi-karoto/pali-transliteration"
)

var scheme = map[string]func(string) string{ //
	"latin🠮latin": func(s string) string { return s },
	"latin🠮thai2": func(s string) string { return palitrans.RomanToThai(strings.ReplaceAll(s, "’", "")) },
	"latin🠮kana":  func(s string) string { return pli.LatinToKana(s) },

	"thai1🠮latin": func(s string) string { return pli.ThaiToLatin(s, 1) },
	"thai1🠮thai2": func(s string) string { return "Not available: everyday thai is a lossy 'encoding' of pali!" },
	"thai1🠮kana":  func(s string) string { return pli.LatinToKana(pli.ThaiToLatin(s, 1)) },

	"thai2🠮latin": func(s string) string { return pli.ThaiToLatin(s, 2) },
	"thai2🠮thai2": func(s string) string { return s },
	"thai2🠮kana":  func(s string) string { return pli.LatinToKana(pli.ThaiToLatin(s, 2)) },
}

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func processTextHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("Received text processing request")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Error reading request body")
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	//inputText := string(body)
	//fmt.Println("resp=", inputText)
	var data struct {
		Text string `json:"text"`
		In   string `json:"inputSelection"`
		Out  string `json:"outputSelection"`
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Error().Err(err).Msg("Invalid JSON data")
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}
	//pp.Println(data)
	var processedText string
	if f, ok := scheme[data.In+"🠮"+data.Out]; ok {
		processedText = f(data.Text)
	} else {
		processedText = "func not found"
	}
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	log.Debug().Msg("Processed text.")
	fmt.Fprintf(w, processedText)
}

func indexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	exeDir, err := os.Executable()
	if err != nil {
		log.Fatal().Err(err).Msg("Error getting executable path")
	}
	templatePath := filepath.Join(filepath.Dir(exeDir), "index.html")

	log.Info().Msgf("Serving index.html from: %s", templatePath)

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func main() {
	router := httprouter.New()
	router.GET("/", indexHandler)
	router.HandlerFunc("POST", "/process", processTextHandler)
	fmt.Println("Server running, exposing http://localhost:8080/")
	log.Fatal().Err(http.ListenAndServe(":8080", router))
}
