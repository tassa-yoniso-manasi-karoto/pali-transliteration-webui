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



var ( 
	scheme = map[string]func(string) string{
		"latin🠮latin": func(s string) string { return s },
		"latin🠮thai2": func(s string) string { return palitrans.RomanToThai(strings.ReplaceAll(s, "’", "")) },

		"thai1🠮latin": func(s string) string { return pli.ThaiToLatin(s, 1) },
		"thai1🠮thai2": func(s string) string { return "Not available: colloquial thai is a lossy encoding of pali!" },

		"thai2🠮latin": func(s string) string { return pli.ThaiToLatin(s, 2) },
		"thai2🠮thai2": func(s string) string { return s },
	}
	index string = `<!DOCTYPE html>
<html>
<head>
  <title>Pali Transliteration</title>
  <style>
    * { font-size: 115%; }
    .container { display: flex; }
    .input-group, .output-group {
      display: flex;
      flex-direction: column; 
      width: 50%;
    }
    select, textarea { 
      width: 100%;
      padding: 15px;
      box-sizing: border-box;
    }

    textarea { height: 100vh; resize: none; } 

    select {
      text-align: center;
      font-weight: bold;
      height: auto;
      min-height: 2.5em;
      align-self: stretch;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="input-group">
      <select id="inputSelect">
        <option value="latin">Latin/Roman</option>
        <option value="thai1">Thai (Colloquial) // อักษรไทย</option>
        <option value="thai2">Thai (Pintu) // แบบพินทุ</option>
      </select>
      <textarea spellcheck="false" id="inputTextArea"></textarea>
    </div>
    <div class="output-group">
      <select id="outputSelect">
        <option value="latin">Latin/Roman</option>
        <option value="thai2">Thai (Pintu) // แบบพินทุ</option>
      </select>
      <textarea spellcheck="false" id="outputTextArea" readonly></textarea>
    </div>
  </div>

  <script>
const inputSelect = document.getElementById("inputSelect");
const outputSelect = document.getElementById("outputSelect");
const inputTextArea = document.getElementById("inputTextArea");
const outputTextArea = document.getElementById("outputTextArea");
const apiEndpoint = "http://localhost:8080/process";

async function callAPI() {
    const inputValue = inputTextArea.value;
    const inputSelection = inputSelect.value;
    const outputSelection = outputSelect.value;
    console.log(inputSelection+"🠮"+outputSelection);
    try {
        const response = await fetch(apiEndpoint, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                text: inputValue,
                inputSelection: inputSelection,
                outputSelection: outputSelection
            })
        });
        if (!response.ok) throw new Error("API request failed 🤖");
        const processedText = await response.text();
        outputTextArea.value = processedText;
    } catch (error) {
        console.error(error);
        outputTextArea.value = "Couldn't reach server";
    }
}
inputTextArea.addEventListener("input", callAPI);
inputSelect.addEventListener("change", callAPI);
outputSelect.addEventListener("change", callAPI);
  </script>
</body>
</html>`
)

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
		log.Error().Err(err).Msg("Error getting executable path")
	}
	templatePath := filepath.Join(filepath.Dir(exeDir), "index.html")
	tmpl := template.New("index")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		log.Info().Msg("Serving built-in index.html")
		tmpl, err = tmpl.Parse(index)
	} else {
		log.Info().Msgf("Serving index.html from: %s", templatePath)
		tmpl, err = template.ParseFiles(templatePath)
	}
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
