# Project 1

this project is a basic sequential web crawler and indexer written in go for 
469.
it starts from a list of seed urls, visits pages one at a time, extracts words,
and writes an inverted index to a csv file.

## files

- `main.go`: main sequential crawler
- `seed-urls.txt`: input file with starting urls
- `visited-site-log.txt`: log of visited urls
- `increment-time-log.txt`: milestone timing log
- `inverted-index.csv`: inverted index output
- `first-10-keywords-log.txt`: first 10 keywords alphabetically

## how to run

1. put your starting urls in `seed-urls.txt`, one url per line
2. set the variable`SITE_LIMIT` in `main.go` to the number of sites you want to crawl
3. run:

```bash
go run main.go
```

## notes

- the crawler is sequential and does not use threads
- duplicate urls are skipped
- words are lowercased, punctuation is removed, and stop words are removed
- the inverted index is written incrementally to the csv during the crawl to reduce memory use
- The completed index is very large (over 50GB), the zstandard compressed reverse index csv can be found here:
```
https://drive.google.com/drive/folders/1Zw-Z1wQtXtB88cPRGJEpuZFgfdMFjGOW?usp=sharing
```
