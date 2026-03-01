package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

const hoursPerYear = 8760.0

type InputData struct {
	LambdaLine         float64
	RepairLineHours    float64
	LambdaTransformer  float64
	RepairTransfHours  float64
	LoadPowerMW        float64
	CostPerMWh         float64
}

type ResultData struct {
	ULineSingle   float64
	ULineDouble   float64
	UTransformer  float64
	USingleSystem float64
	UDoubleSystem float64
	ASingleSystem float64
	ADoubleSystem float64

	EENSSingleMWh float64
	EENSDoubleMWh float64
	CostSingle    float64
	CostDouble    float64
}

type PageData struct {
	Input  InputData
	Result *ResultData
}

var tpl *template.Template

func main() {
	var err error
	tpl, err = template.ParseFiles(
		filepath.Join("templates", "layout.html"),
		filepath.Join("templates", "index.html"),
	)
	if err != nil {
		log.Fatalf("cannot parse templates: %v", err)
	}

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", indexHandler)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := PageData{
			Input: defaultInput(),
		}
		if err := tpl.ExecuteTemplate(w, "layout", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "cannot parse form", http.StatusBadRequest)
			return
		}

		input := InputData{
			LambdaLine:         parseFloat(r, "lambdaLine", 0.3),
			RepairLineHours:    parseFloat(r, "repairLine", 10),
			LambdaTransformer:  parseFloat(r, "lambdaTransformer", 0.15),
			RepairTransfHours:  parseFloat(r, "repairTransformer", 20),
			LoadPowerMW:        parseFloat(r, "loadPower", 10),
			CostPerMWh:         parseFloat(r, "costPerMWh", 1000),
		}

		result := calculate(input)

		data := PageData{
			Input:  input,
			Result: &result,
		}

		if err := tpl.ExecuteTemplate(w, "layout", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func defaultInput() InputData {
	return InputData{
		LambdaLine:        0.3,
		RepairLineHours:   10,
		LambdaTransformer: 0.15,
		RepairTransfHours: 20,
		LoadPowerMW:       10,
		CostPerMWh:        1000,
	}
}

func parseFloat(r *http.Request, name string, def float64) float64 {
	raw := strings.TrimSpace(r.FormValue(name))
	if raw == "" {
		return def
	}
	raw = strings.ReplaceAll(raw, ",", ".")
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return def
	}
	return value
}

func calculate(in InputData) ResultData {
	uLine := in.LambdaLine * in.RepairLineHours / hoursPerYear
	uLineDouble := uLine * uLine
	uTransformer := in.LambdaTransformer * in.RepairTransfHours / hoursPerYear

	uSingleSystem := 1 - (1-uLine)*(1-uTransformer)
	uDoubleSystem := 1 - (1-uLineDouble)*(1-uTransformer)

	aSingleSystem := 1 - uSingleSystem
	aDoubleSystem := 1 - uDoubleSystem

	eensSingle := uSingleSystem * hoursPerYear * in.LoadPowerMW
	eensDouble := uDoubleSystem * hoursPerYear * in.LoadPowerMW

	costSingle := eensSingle * in.CostPerMWh
	costDouble := eensDouble * in.CostPerMWh

	return ResultData{
		ULineSingle:   uLine,
		ULineDouble:   uLineDouble,
		UTransformer:  uTransformer,
		USingleSystem: uSingleSystem,
		UDoubleSystem: uDoubleSystem,
		ASingleSystem: aSingleSystem,
		ADoubleSystem: aDoubleSystem,
		EENSSingleMWh: eensSingle,
		EENSDoubleMWh: eensDouble,
		CostSingle:    costSingle,
		CostDouble:    costDouble,
	}
}

