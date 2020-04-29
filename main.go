package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
	//"io/ioutil"
)

type Source struct {
	ID		interface{} `json:"id"`
	Name 	string 		`json:"name"`
}

type Article struct {
	Source		Source		`json:"source"`
	Author		string		`json:"author"`
	Title		string		`json:"title"`
	Description	string		`json:"description"`
	URL			string		`json:"url"`
	URLToImage	string		`json:"urlToImage"`
	PublishedAt	time.Time	`json:"publishedAt"`
	Content		string		`json:"content"`
}

type Results struct {
	Status			string		`json:"status"`
	TotalResults	int			`json:"totalResults"`
	Articles		[]Article	`json:"articles"`
}

type Search struct {
	SearchKey	string
	NextPage	int
	TotalPages	int
	Results		Results
}

type NewsAPIError struct {
	Status 	string	`json:"status"`
	Code 	string	`json:"code"`
	Message	string	`json:"message"`
}

func (s *Search) IsLastPage() bool {
	return s.NextPage >= s.TotalPages
}

func (s *Search) CurrentPage() int {
	if s.NextPage == 1 {
		return s.NextPage
	}
	return s.NextPage - 1
}

func (s *Search) PreviousPage() int {
	return s.CurrentPage() - 1
}

var apiKey *string
var tmpl = template.Must(template.ParseFiles("index.html"))

func (a *Article) FormatPublishedDate() string {
	year, month, day := a.PublishedAt.Date()
	return fmt.Sprintf("%v %d, %d", month, day, year)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl.Execute(w, nil)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.URL.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	params := u.Query()
	searchKey := params.Get("q")
	page := params.Get("page")
	if page == "" {
		page = "1"
	}
	
	fmt.Println("Search Query is: ", searchKey)
	fmt.Println("Results page is: ", page)
	
	search := &Search{}
	search.SearchKey = searchKey
	
	next, err := strconv.Atoi(page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	search.NextPage = next
	pageSize := 20
	
	endpoint := fmt.Sprintf("https://newsapi.org/v2/everything?q=%s&pageSize=%d&page=%d&apiKey=%s&sortBy=publishedAt&language=en", url.QueryEscape(search.SearchKey), pageSize, search.NextPage, *apiKey)
	resp, err := http.Get(endpoint)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		newError := &NewsAPIError{}
		err := json.NewDecoder(resp.Body).Decode(newError)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Error(w, newError.Message, http.StatusInternalServerError)
		return
	}
	
	//body, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(body))
	
	err = json.NewDecoder(resp.Body).Decode(&search.Results)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	search.TotalPages = int(math.Ceil(float64(search.Results.TotalResults / pageSize)))
	
	if ok := !search.IsLastPage(); ok {
		search.NextPage++
	}
	
	err = tmpl.Execute(w, search)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	apiKey = flag.String("apiKey", "", "8832a737dc9f4653833505220021d32b")
	flag.Parse()
	
	if *apiKey == "" {
		log.Fatal("apiKey must be set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	
	fs := http.FileServer(http.Dir("resources"))
	http.Handle("/resources/", http.StripPrefix("/resources/", fs))
	
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/search", searchHandler)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}