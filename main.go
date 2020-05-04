package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

const (
	wordSource  = "https://www.ceskenoviny.cz"
	wordMeaning = "https://slovniky.lingea.cz/anglicko-cesky/"
	telegramApi = "https://api.telegram.org/bot"
)

var (
	telegramToken  = os.Getenv("TELEGRAM_TOKEN")   // your bot token
	telegramUserID = os.Getenv("TELEGRAM_USER_ID") // user id or username
)

type Definition struct {
	Word         string
	Translations []string
	Examples     []string
}

type DefinitionNotFound struct{}

func (DefinitionNotFound) Error() string {
	return "definition not found!"
}

func main() {
	if telegramToken == "" || telegramUserID == "" {
		panic("the environment variables are not set")
	}
	articleUrl, err := getArticleUrl()
	if err != nil {
		log.Fatal(err)
	}
	text, err := getArticleText(articleUrl)
	if err != nil {
		log.Fatal(err)
	}
	var word string
	var definition *Definition
	for {
		word = getRandomWord(text)
		definition, err = getDefinition(word)
		if err != nil {
			if dnf, ok := err.(DefinitionNotFound); ok {
				log.Println(dnf.Error())
				continue
			}
			log.Fatal(err)
		}
		break
	}
	message := createMessage(definition)
	err = sendMessage(message)
	if err != nil {
		log.Fatal(err)
	}
	err = sendMessage(text)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(definition.Word + " sent!")
}

// Returns a link to the first article
func getArticleUrl() (string, error) {
	req, err := http.Get(wordSource)
	if err != nil {
		return "", err
	}
	defer req.Body.Close()

	doc, err := goquery.NewDocumentFromReader(req.Body)
	if err != nil {
		return "", err
	}

	a := doc.Find("#pravevydano ul li a")
	href, exists := a.Attr("href")
	if exists {
		return href, nil
	}
	return "", errors.New("no href attribute")
}

// Parses the first paragraph of the article
func getArticleText(url string) (string, error) {
	req, err := http.Get(wordSource + url)
	if err != nil {
		return "", err
	}
	if req.StatusCode != 200 {
		return "", errors.New(req.Status)
	}
	defer req.Body.Close()
	doc, err := goquery.NewDocumentFromReader(req.Body)
	if err != nil {
		return "", err
	}
	p := doc.Find("#articlebody p.big")
	return p.Text(), nil
}

// Returns random word from the given text
func getRandomWord(text string) string {
	arr := strings.Fields(text)
	rand.Seed(time.Now().UTC().UnixNano())
	for {
		i := rand.Intn(len(arr))
		if len(arr[i]) > 3 && !unicode.IsUpper(rune(arr[i][0])) {
			return arr[i]
		}
	}
}

// Makes a request to an online dictionary and parses the word definition
func getDefinition(word string) (*Definition, error) {
	req, err := http.Get(wordMeaning + word)
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()
	doc, err := goquery.NewDocumentFromReader(req.Body)
	if err != nil {
		return nil, err
	}
	table := doc.Find("table.entry")
	if table == nil {
		return nil, DefinitionNotFound{}
	}
	meaning := &Definition{}
	meaning.Word = table.Find("h1.lex_ful_entr").Text()
	table.Find("span.lex_ful_tran").Each(func(i int, selection *goquery.Selection) {
		meaning.Translations = append(meaning.Translations, selection.Text())
	})
	table.Find("span.lex_ful_samp2").Each(func(i int, selection *goquery.Selection) {
		meaning.Examples = append(meaning.Examples, selection.Text())
	})
	table.Find("span.lex_ful_coll2").Each(func(i int, selection *goquery.Selection) {
		meaning.Examples = append(meaning.Examples, selection.Text())
	})
	return meaning, nil
}

// Creates message in HTML-format for telegram
func createMessage(definition *Definition) string {
	builder := strings.Builder{}
	//emojis := [10]string{
	//	":one:", ":two:", ":three:", ":four:", ":five:",
	//	":six:", ":seven:", ":eight:", ":nine:", ":keycap_ten:",
	//}
	builder.WriteString(fmt.Sprintf("<b>%s</b>\n\n", definition.Word))
	for i, translate := range definition.Translations {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, translate))
	}
	if len(definition.Examples) != 0 {
		builder.WriteString("\n<b>Examples</b>:\n")
		for i, example := range definition.Examples {
			builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, example))
		}
	}
	builder.WriteString("\nSource:⤵️")
	return builder.String()
}

// no comments
func sendMessage(message string) error {
	query := fmt.Sprintf("chat_id=%s&text=%s&parse_mode=HTML", telegramUserID, message)
	query = url.PathEscape(query)
	req, err := http.Get(telegramApi + telegramToken + "/sendMessage?" + query)
	if err != nil {
		return err
	}
	defer req.Body.Close()
	if req.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(req.Body)
		return errors.New(string(body))
	}
	return nil
}
