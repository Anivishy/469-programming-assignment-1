package main

import (
	"bufio"
	//"fmt"
	"encoding/csv"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

//consts
const (
	SITE_LIMIT      = 1000
	REQUEST_TIMEOUT = 10 * time.Second
	USER_AGENT      = "DS469Crawler"
)

//global vars
var parsed_urls int = 0
var seen_sites map[string]struct{}

var stop_words = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {},
	"be": {}, "but": {}, "by": {}, "for": {}, "if": {}, "in": {},
	"into": {}, "is": {}, "it": {}, "no": {}, "not": {}, "of": {},
	"on": {}, "or": {}, "such": {}, "that": {}, "the": {}, "their": {},
	"then": {}, "there": {}, "these": {}, "they": {}, "this": {},
	"to": {}, "was": {}, "will": {}, "with": {},
}

var (
	reScript = regexp.MustCompile(`(?is)<script.*?>.*?</script>`)
	reStyle  = regexp.MustCompile(`(?is)<style.*?>.*?</style>`)
	reTags   = regexp.MustCompile(`(?is)<[^>]*>`)
	rePunc   = regexp.MustCompile(`[^a-z0-9\s]+`)
	reHref   = regexp.MustCompile(`(?i)href\s*=\s*["']([^"'#]+)["']`)
	reWord   = regexp.MustCompile(`^[a-z]+$`)
	httpClient = &http.Client{
		Timeout: REQUEST_TIMEOUT,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}
)

func main() {
	seen_sites = make(map[string]struct{})
	//reading in seed urls file
	urls, err := os.ReadFile("seed-urls.txt")

	if err != nil {
		panic("Error when trying to read in seed file : " + err.Error())
	}


	//parsing seed urls
	url_queue := []string{}
	for _, rawURL := range strings.FieldsFunc(string(urls), func(r rune) bool {
		return r == '\r' || r == '\n'
	}) {
		candidate := strings.TrimSpace(rawURL)
		if candidate == "" {
			continue
		}
		if enqueue_if_new(candidate) {
			url_queue = append(url_queue, candidate)
		}
	}

	//init for both log files and inverse index csv file
	visited_site_log, err := os.OpenFile("visited-site-log.txt", os.O_WRONLY|os.O_CREATE|
		os.O_TRUNC, 0644)
	if err != nil{
		panic("Error initilizing visited sites log file.")
	}
	defer visited_site_log.Close()
	visited_site_writer := bufio.NewWriter(visited_site_log)
	defer visited_site_writer.Flush()

	increment_time_log, err := os.OpenFile("increment-time-log.txt", os.O_WRONLY|os.O_CREATE|
		os.O_TRUNC, 0644)

	if err != nil{
		panic("Error initilizing increment time log file.")
	}
	defer increment_time_log.Close()
	increment_time_writer := bufio.NewWriter(increment_time_log)
	defer increment_time_writer.Flush()

	index_file, err := os.Create("inverted-index.csv")
	if err != nil {
		panic("Error creating index csv file: " + err.Error())
	}
	defer index_file.Close()
	index_buffered_file := bufio.NewWriter(index_file)
	defer index_buffered_file.Flush()

	index_writer := csv.NewWriter(index_buffered_file)
	defer index_writer.Flush()

	err = index_writer.Write([]string{"word", "url"})
	if err != nil {
		panic("Error writing csv header: " + err.Error())
	}

	first_keywords := []string{}
	first_keyword_set := make(map[string]struct{})

	startTime := time.Now()
	queue_index := 0

	for (parsed_urls < SITE_LIMIT) {
		if queue_index >= len(url_queue) {
			panic("url list is empty")
		}
		cur_url := url_queue[queue_index]
		queue_index++

		//shrink the queue slice once consumed a large prefix
		if queue_index >= 1024 && queue_index*2 >= len(url_queue) {
			url_queue = append([]string(nil), url_queue[queue_index:]...)
			queue_index = 0
		}

		page := fetch_page(cur_url)
		if page == "" {
			continue
		}
		unique_words := index_site(cur_url, page, index_writer)
		first_keywords = track_first_n_keywords(first_keywords, first_keyword_set, unique_words, 10)

		//log site visit, update sites list and increment count of visited 
		// urls
		log_site("Visited New Site: ", cur_url, visited_site_writer)
		url_queue = append_new_sites_to_queue(url_queue, get_sites(cur_url, page))
		parsed_urls ++

		//buffered write for indexes to solve our issue of holding all the 
		// indexes in RAM and computer crashing
		if parsed_urls%5000 == 0 {
			index_writer.Flush()
			index_buffered_file.Flush()
		}

		log_milestone_time(parsed_urls, startTime, increment_time_writer)
	}

	totalDuration := time.Since(startTime)
	log_site("Total time for " + strconv.Itoa(SITE_LIMIT) + " sites: ", 
				totalDuration.String(), visited_site_writer)

	write_first_n_keywords("first-10-keywords-log.txt", first_keywords)
}

func enqueue_if_new(c_url string) bool {
	if _, ok := seen_sites[c_url]; ok {
		return false
	}

	seen_sites[c_url] = struct{}{}
	return true
}

func append_new_sites_to_queue(url_queue []string, sites []string) []string {
	for _, site := range sites {
		if enqueue_if_new(site) {
			url_queue = append(url_queue, site)
		}
	}

	return url_queue
}

func fetch_page(c_url string) string {
	req, err := http.NewRequest(http.MethodGet, c_url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", USER_AGENT)

	resp, err := httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return ""
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if contentType != "" &&
		!strings.Contains(contentType, "text/html") &&
		!strings.Contains(contentType, "application/xhtml+xml") &&
		!strings.Contains(contentType, "text/plain") {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return string(body)
}

func index_site(c_url string, page string, index_writer *csv.Writer) []string {
	//inverted index every word in the site, with the key being the word and the
	// originating url (c_url in this case)
	unique_words := normalize_and_get_unique_words(page)
	write_index_rows(index_writer, c_url, unique_words)
	return unique_words
}

func normalize_and_get_unique_words(page string) []string{
	//helper method to index_site that parses the fetched page content, gets all
	// the unique normalized words present, and returns them as a list

	// remove script/style blocks first
	page = reScript.ReplaceAllString(page, " ")
	page = reStyle.ReplaceAllString(page, " ")

	//remove html tags
	page = reTags.ReplaceAllString(page, " ")

	//lowercase
	page = strings.ToLower(page)

	//remove punctuation / non alphabet chars
	page = rePunc.ReplaceAllString(page, " ")

	//split into words
	words := strings.Fields(page)

	//collect unique non-stop words
	unique_map := make(map[string]bool)
	for _, word := range words {
		if word == "" {
			continue
		}
		if !reWord.MatchString(word) {
			continue
		}
		if _, isStopWord := stop_words[word]; isStopWord {
			continue
		}
		unique_map[word] = true
	}

	unique_words := make([]string, 0, len(unique_map))
	for word := range unique_map {
		unique_words = append(unique_words, word)
	}

	return unique_words
}

func get_sites(c_url string, page string) []string{
	//returns a list of sites found in the fetched content for the current site
	// find href links
	matches := reHref.FindAllStringSubmatch(page, -1)

	base_url, err := url.Parse(c_url)
	if err != nil {
		return nil
	}

	found_sites := []string{}
	seen_links := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		link := strings.TrimSpace(match[1])
		if link == "" {
			continue
		}

		parsed_link, err := url.Parse(link)
		if err != nil {
			continue
		}

		absolute_link := base_url.ResolveReference(parsed_link).String()

		// only keep http/https links
		if !strings.HasPrefix(absolute_link, "http://") && !strings.HasPrefix(absolute_link, "https://") {
			continue
		}

		if !seen_links[absolute_link] {
			found_sites = append(found_sites, absolute_link)
			seen_links[absolute_link] = true
		}
	}

	return found_sites
}

func log_site(message_prefix string, c_url string, write_file io.Writer){
	//write the message_prefix + c_url string combination to the logfile
	_, _ = write_file.Write([]byte(message_prefix + c_url + "\n"))
}

func log_milestone_time(count int, startTime time.Time, write_file io.Writer) {
	switch count {
	case 10, 100, 1000, 10000, 100000, 1000000:
		elapsed := time.Since(startTime)
		log_site("Time for "+strconv.Itoa(count)+" sites: ", elapsed.String(), write_file)
	}
}

func write_index_rows(writer *csv.Writer, c_url string, unique_words []string) {
	for _, word := range unique_words {
		err := writer.Write([]string{word, c_url})
		if err != nil {
			panic("Error writing csv row: " + err.Error())
		}
	}
}

func track_first_n_keywords(first_keywords []string, first_keyword_set map[string]struct{}, words []string, count int) []string {
	//helper method that keeps only the first n keywords alphabetically
	// withou storing every keyword we ever see in memory
	for _, word := range words {
		//skip words we are already tracking
		if _, exists := first_keyword_set[word]; exists {
			continue
		}

		//fill up the list until we have count tracked words
		if len(first_keywords) < count {
			first_keywords = append(first_keywords, word)
			sort.Strings(first_keywords)
			first_keyword_set[word] = struct{}{}
			continue
		}

		//the slice stays sorted so the last word is the current largest one
		largest_tracked := first_keywords[len(first_keywords)-1]

		//if the new word comes after the current largest tracked word,
		// it cannot be part of the first n words
		if word >= largest_tracked {
			continue
		}

		//otherwise replace the current largest tracked word and re-sort
		delete(first_keyword_set, largest_tracked)
		first_keywords = first_keywords[:len(first_keywords)-1]
		first_keywords = append(first_keywords, word)
		sort.Strings(first_keywords)
		first_keyword_set[word] = struct{}{}
	}

	return first_keywords
}

func write_first_n_keywords(filename string, keywords []string) {
	file, err := os.Create(filename)
	if err != nil {
		panic("Error creating keyword log file: " + err.Error())
	}
	defer file.Close()
	buffered_file := bufio.NewWriter(file)
	defer buffered_file.Flush()

	for _, word := range keywords {
		_, _ = buffered_file.WriteString(word + "\n")
	}
}
