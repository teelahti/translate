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
	"slices"
	"strings"
	"time"

	translate "cloud.google.com/go/translate/apiv3"
	"cloud.google.com/go/translate/apiv3/translatepb"

	"github.com/fatih/color"
)

const EN = "en-US"

var httpClient = &http.Client{Timeout: 5 * time.Second}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printHelpAndExit()
	}

	if len(args) == 1 {
		printDictionary(args[0])
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

		if to == EN {
			printDictionary(s.TranslatedText)
		}
	}
}

func printHelpAndExit() {
	fmt.Println("Arguments missing. Supports two modes:")
	fmt.Println("    fromLang toLang term")
	fmt.Println("    term")
	fmt.Println("Language codes use BCP-47 format: fi-FI, en-US, de-DE, etc.")
	os.Exit(1)
}

// printDictionary looks up an English word using the Free Dictionary API
// (freedictionaryapi.com, Wiktionary data) and prints definitions,
// phonetics, synonyms, and antonyms.
func printDictionary(w string) {
	apiURL := fmt.Sprintf("https://freedictionaryapi.com/api/v1/entries/en/%s", url.PathEscape(strings.ToLower(w)))

	resp, err := httpClient.Get(apiURL)
	if err != nil {
		fmt.Println("Dictionary request failed:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read dictionary response:", err)
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		printSuggestions(w)
		return
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Dictionary API error (HTTP %d)\n", resp.StatusCode)
		return
	}

	var result DictionaryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("Failed to parse dictionary response:", err)
		return
	}

	// Unknown words come back as 200 with no entries
	if len(result.Entries) == 0 {
		printSuggestions(w)
		return
	}

	if phonetic := findPhonetic(result.Entries); phonetic != "" {
		fmt.Println(phonetic)
	}

	// Print definitions grouped by part of speech
	const maxMeanings = 3
	const maxDefs = 2

	var allSyns, allAnts []string

	for i, entry := range result.Entries {
		if i >= maxMeanings {
			break
		}

		fl := fmt.Sprintf("%-4.4s", entry.PartOfSpeech)
		flPrinter := color.New(color.FgMagenta).PrintFunc()

		for j, sense := range entry.Senses {
			if j >= maxDefs {
				break
			}

			if j == 0 {
				flPrinter(fl)
			} else {
				fmt.Print("    ")
			}

			fmt.Printf(" %s\n", sense.Definition)
		}

		allSyns = append(allSyns, entry.Synonyms...)
		allAnts = append(allAnts, entry.Antonyms...)
		for _, sense := range entry.Senses {
			allSyns = append(allSyns, sense.Synonyms...)
			allAnts = append(allAnts, sense.Antonyms...)
		}
	}

	allSyns = dedupe(allSyns)
	allAnts = dedupe(allAnts)

	// Print aggregated synonyms and antonyms
	const synLimit = 6
	printWordList(allSyns, "syn.:", synLimit)
	printWordList(allAnts, "ant.:", synLimit)

	fmt.Println()
}

// printSuggestions fetches spelling suggestions from Datamuse when a word
// is not found in the dictionary.
func printSuggestions(w string) {
	apiURL := fmt.Sprintf("https://api.datamuse.com/words?sl=%s&max=5", url.QueryEscape(w))

	resp, err := httpClient.Get(apiURL)
	if err != nil {
		fmt.Println("not found in dictionary")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("not found in dictionary")
		return
	}

	var suggestions []struct {
		Word string `json:"word"`
	}

	if err := json.Unmarshal(body, &suggestions); err != nil || len(suggestions) == 0 {
		fmt.Println("not found in dictionary")
		return
	}

	words := make([]string, len(suggestions))
	for i, s := range suggestions {
		words[i] = s.Word
	}

	fmt.Println("not found; try:", strings.Join(words, ", "))
}

// printWordList prints a labeled list of words (for synonyms/antonyms).
func printWordList(words []string, prefix string, limit int) {
	if len(words) == 0 {
		return
	}

	if len(words) > limit {
		words = words[:limit]
	}

	prefPrinter := color.New(color.FgCyan).PrintfFunc()

	fmt.Println()
	prefPrinter("     %s ", prefix)
	fmt.Println(strings.Join(words, ", "))
}

// findPhonetic returns the IPA transcription, preferring the US one.
func findPhonetic(entries []DictionaryEntry) string {
	var first string
	for _, e := range entries {
		for _, p := range e.Pronunciations {
			if p.Type != "ipa" || p.Text == "" {
				continue
			}
			if slices.Contains(p.Tags, "US") {
				return p.Text
			}
			if first == "" {
				first = p.Text
			}
		}
	}
	return first
}

// dedupe removes duplicate words while keeping the original order.
func dedupe(words []string) []string {
	seen := make(map[string]bool, len(words))
	var out []string
	for _, w := range words {
		if !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	return out
}

func getGcpParent() string {
	const GCP_ENV = "TRANSLATE_GCP_PARENT"
	return getSecret(GCP_ENV, "GCP Parent identifier missing (like projects/my-project).")
}

func getSecret(env string, desc string) string {
	envf := env + "_FILE"

	// Option 1: secret in env variable
	val, found := os.LookupEnv(env)
	if found {
		return val
	}

	// Option 2: env variable tells the location of the secret file
	if fp, found := os.LookupEnv(envf); found {
		b, err := os.ReadFile(fp)
		if err == nil {
			return strings.SplitN(string(b), "\n", 2)[0]
		}
	}

	fmt.Println(desc, "The env variable", env, "should contain the configuration value, or", envf, "the path to the file containing the configuration value on first line.")
	os.Exit(1)

	return ""
}

// Free Dictionary API types (freedictionaryapi.com)

type DictionaryResponse struct {
	Word    string            `json:"word"`
	Entries []DictionaryEntry `json:"entries"`
}

type DictionaryEntry struct {
	PartOfSpeech   string `json:"partOfSpeech"`
	Pronunciations []struct {
		Type string   `json:"type"`
		Text string   `json:"text"`
		Tags []string `json:"tags"`
	} `json:"pronunciations"`
	Senses []struct {
		Definition string   `json:"definition"`
		Examples   []string `json:"examples"`
		Synonyms   []string `json:"synonyms"`
		Antonyms   []string `json:"antonyms"`
	} `json:"senses"`
	Synonyms []string `json:"synonyms"`
	Antonyms []string `json:"antonyms"`
}
