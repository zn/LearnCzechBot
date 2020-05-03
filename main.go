package main

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	url2 "net/url"
	"strings"
	"time"
	"unicode"
)

const wordSource = "https://www.ceskenoviny.cz"
const wordMeaning = "https://slovniky.lingea.cz/anglicko-cesky/"
const telegramApi = "https://api.telegram.org/bot"

const telegramToken = "" // your bot token
const telegramUserID = 000000000 // your user id

type Definition struct{
	Word      string
	Translations []string
	Examples   []string
}

func main(){
	url, err := getArticleUrl()
	if err != nil{
		log.Fatal(err)
	}
	text, err := getArticleText(url)
	if err != nil{
		log.Fatal(err)
	}
	word := getRandomWord(text)
	definition, err := getDefinition(word)
	if err != nil{
		log.Fatal(err)
	}
	message := generateMessage(definition)
	err = sendMessage(message)
	if err != nil{
		log.Fatal(err)
	}
	log.Println(definition.Word + " sent!")
}

// Returns a link to the first article
func getArticleUrl() (string, error){
	req, err:= http.Get(wordSource)
	if err != nil {
		return "", err
	}
	defer req.Body.Close()

	doc, err := goquery.NewDocumentFromReader(req.Body)
	if err != nil{
		return "", err
	}

	a := doc.Find("#pravevydano ul li a")
	href, exists := a.Attr("href")
	if exists{
		return href, nil
	}
	return "", errors.New("no href attribute")
}

// Parses the first paragraph of the article
func getArticleText(url string) (string, error) {
	req, err := http.Get(wordSource + url)
	if err != nil{
		return "", err
	}
	if req.StatusCode != 200{
		return "", errors.New(req.Status)
	}
	defer req.Body.Close()
	doc, err := goquery.NewDocumentFromReader(req.Body)
	if err != nil{
		return "", err
	}
	p := doc.Find("#articlebody p.big")
	return p.Text(), nil
}

// Takes random word from the given text
func getRandomWord(text string) string{
	arr := strings.Fields(text)
	rand.Seed(time.Now().UTC().UnixNano())
	for{
		i := rand.Intn(len(arr))
		if len(arr[i]) > 3 && !unicode.IsUpper(rune(arr[i][0])){
			return arr[i]
		}
	}
}

// Makes a request to a dictionary and parses definition of the word
func getDefinition(word string) (*Definition, error) {
	req, err := http.Get(wordMeaning + word)
	if err != nil{
		return nil, err
	}
	defer req.Body.Close()
	doc, err := goquery.NewDocumentFromReader(req.Body)
	if err != nil{
		return nil, err
	}
	table := doc.Find("table.entry")
	if table == nil{
		return nil, errors.New("word not found")
	}
	meaning := &Definition{}
	meaning.Word =  table.Find("h1.lex_ful_entr").Text()
	table.Find("span.lex_ful_tran").Each(func(i int, selection *goquery.Selection){
		meaning.Translations = append(meaning.Translations, selection.Text())
	})
	table.Find("span.lex_ful_samp2").Each(func(i int, selection *goquery.Selection){
		meaning.Examples = append(meaning.Examples, selection.Text())
	})
	table.Find("span.lex_ful_coll2").Each(func(i int, selection *goquery.Selection){
		meaning.Examples = append(meaning.Examples, selection.Text())
	})
	return meaning, nil
}

// Creates message in HTML-format for telegram
func generateMessage(definition *Definition) string{
	builder := strings.Builder{}
	//emojis := [10]string{
	//	":one:", ":two:", ":three:", ":four:", ":five:",
	//	":six:", ":seven:", ":eight:", ":nine:", ":keycap_ten:",
	//}
	builder.WriteString(fmt.Sprintf("<b>%s</b>\n\n", definition.Word))
	for i, translate := range definition.Translations{
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, translate))
	}
	if len(definition.Examples) != 0 {
		builder.WriteString("\n<b>Examples</b>:\n")
		for i, example := range definition.Examples {
			builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, example))
		}
	}
	return builder.String()
}

// no comments
func sendMessage(message string) error{
	query := fmt.Sprintf("chat_id=%d&text=%s&parse_mode=HTML", telegramUserID, message)
	query = url2.PathEscape(query)
	req, err := http.Get(telegramApi+telegramToken+"/sendMessage?" + query)
	if err != nil{
		return err
	}
	defer req.Body.Close()
	if req.StatusCode != http.StatusOK{
		body, _ := ioutil.ReadAll(req.Body)
		return errors.New(string(body))
	}
	return nil
}