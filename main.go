package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	translate "cloud.google.com/go/translate/apiv3"
	"cloud.google.com/go/translate/apiv3/translatepb"

	"github.com/fatih/color"
)

const EN = "en-US"

func main() {

	args := os.Args[1:]
	if len(args) == 0 {
		printHelpAndExit()
	}

	if len(args) == 1 {
		printThesaurus(args[0])
		os.Exit(0)
	}

	if len(args) < 3 {
		printHelpAndExit()
	}

	from, to, term := args[0], args[1], args[2]

	ctx := context.Background()
	c, err := translate.NewTranslationClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// https://pkg.go.dev/cloud.google.com/go/translate/apiv3/translatepb#TranslateTextRequest
	req := &translatepb.TranslateTextRequest{
		Parent:             getGcpParent(),
		SourceLanguageCode: from,
		TargetLanguageCode: to,
		MimeType:           "text/plain",
		Contents:           []string{term},
	}
	resp, err := c.TranslateText(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < len(resp.Translations); i++ {
		s := resp.Translations[i]

		heading := color.New(color.FgWhite).Add(color.Bold)
		heading.Println(s.TranslatedText)

		if to == "en-US" {
			// Consider adding also Dictionary entries from
			// https://dictionaryapi.com/products/api-collegiate-dictionary
			printThesaurus(s.TranslatedText)
		}
	}
}

func printHelpAndExit() {
	fmt.Println("Arguments missing. Supports two modes: \n    fromLang toLang term\n    term")
	os.Exit(1)
}

func printThesaurus(w string) {

	tmpl := "https://www.dictionaryapi.com/api/v3/references/thesaurus/json/%s?key=%s"
	apiUrl := fmt.Sprintf(tmpl, url.QueryEscape(w), getThApiKey())

	resp, err := http.Get(apiUrl)
	if err != nil {
		fmt.Println("No response from request")
		os.Exit(1)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body) // response body is []byte

	var result ThesaurusResponse
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to the go struct pointer
		fmt.Println("Can not unmarshal JSON")
		os.Exit(2)
	}

	for _, r := range result {
		fmt.Println()

		for _, sf := range r.Shortdef {
			fmt.Println(sf)
		}

		flPrinter := color.New(color.FgMagenta).PrintFunc()
		flPrinter(r.Fl[:4])
		p := color.New(color.FgWhite).PrintfFunc()
		p(" %s\n", strings.Join(r.Meta.Stems, ", "))

		printSyns(r.Meta.Syns, "syn.:", 5)
		printSyns(r.Meta.Ants, "ant.:", 5)

	}
}

func printSyns(lst [][]string, prefix string, limit int) {

	prefPrinter := color.New(color.FgCyan).PrintfFunc()

	// Print first N synonyms or antonyms since these lists can be huge
	if len(lst) > 0 {
		fmt.Println()

		for _, synGrp := range lst {
			prefPrinter("     %s ", prefix)

			for j, syn := range synGrp {
				fmt.Print(syn)

				if j > limit {
					break
				}

				fmt.Print(", ")
			}

			fmt.Println()
		}
	}
}
func getGcpParent() string {
	const GCP_ENV = "TRANSLATE_GCP_PARENT"
	parent, found := os.LookupEnv(GCP_ENV)
	if !found {
		fmt.Println("GCP Parent identifier missing (like projects/my-project). Add it with env", GCP_ENV)
		os.Exit(1)
	}
	return parent
}

func getThApiKey() string {
	const TH_API_ENV = "TRANSLATE_THESAURUS_API_KEY"
	th_api_key, found := os.LookupEnv(TH_API_ENV)
	if !found {
		fmt.Println("Merriam-Webster thesaurus API key missing. Add it with env", TH_API_ENV)
		os.Exit(1)
	}
	return th_api_key
}

// Generated from Thesaurus API response with
// https://mholt.github.io/json-to-go/
type ThesaurusResponse []struct {
	Meta struct {
		ID      string `json:"id"`
		UUID    string `json:"uuid"`
		Src     string `json:"src"`
		Section string `json:"section"`
		Target  struct {
			Tuuid string `json:"tuuid"`
			Tsrc  string `json:"tsrc"`
		} `json:"target"`
		Stems     []string   `json:"stems"`
		Syns      [][]string `json:"syns"`
		Ants      [][]string `json:"ants"`
		Offensive bool       `json:"offensive"`
	} `json:"meta"`
	Hwi struct {
		Hw string `json:"hw"`
	} `json:"hwi"`
	Fl  string `json:"fl"`
	Def []struct {
		Sseq [][][]any `json:"sseq"`
	} `json:"def"`
	Shortdef []string `json:"shortdef"`
}
