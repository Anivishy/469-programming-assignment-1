package main

import (
	"fmt"
	"os"
	//"bytes"
	"strings"
	//"io"
	//"net/http"
)

//consts
const (
	SITE_LIMIT = 10000000
)

//global vars
var parsed_urls int = 0
var visited_sites map[string]string
var inverted_index map[string][]string

func main() {
	//reading in seed urls file
	urls, err := os.ReadFile("seed-urls.txt")

	if err != nil {
		panic("Error when trying to read in seed file : " + err.Error())
	}

	urls_strs := strings.Split(strings.TrimSpace(string(urls)), "\r\n")

	f, err := os.OpenFile("visited-site-log.txt", os.O_WRONLY | os.O_CREATE |
	 						os.O_WRONLY, 0644)
	if err != nil{
		panic("Error initilizing visited sites log file.")
	}
	defer f.Close()

	for (parsed_urls < SITE_LIMIT) {
		if len(urls_strs) == 0{
			panic("url list is empty")
		}
		cur_url := urls_strs[0]
		urls_strs = urls_strs[1:]
		index_site(cur_url)

		//log site visit, update sites list and increment count of visited 
		// urls
		log_site("Visited New Site: ", cur_url, f)
		urls_strs = append(urls_strs, get_sites(cur_url)...)
		parsed_urls ++
	}
}

func index_site(c_url string) {
	//inverted index every word in the site, with the key being the word and the
	// originating url (c_url in this case)
	unique_words := normalize_and_get_unique_words(c_url)
	for _, val := range(unique_words){
		inverted_index[val] = append(inverted_index[val], val)
	}
}

func normalize_and_get_unique_words(c_url string) []string{
	//helper method to index_site that parses the entire text of the given 
	// website c_url, gets all the unique words present normalized and returns a
	// list of these words
	return nil //[]string{}
}

func get_sites(c_url string) []string{
	//returns a list of sites found on the current site (c_url)
	// use validate_url to filter out already visited sites as they are parsed 
	// in. Also make sure to update validate_url with each new url that is read 
	// in
	return nil //[]string{}
}

func validate_url(c_url string) bool{
	//validates if the url has alreayd been visited from the visited sites map
	_, ok := visited_sites[c_url]
	if ok {
		return false
	}
	visited_sites[c_url] = c_url
	return true
}

func log_site(message_prefix string, c_url string, write_file *os.File){
	//write the message_prefix + c_url string combination to the logfile
	write_file.WriteString(message_prefix + c_url + "\n")
}