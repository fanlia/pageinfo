
package main

import (
	"fmt"
	"github.com/qiniu/iconv"
    "github.com/PuerkitoBio/goquery"
    "net/http"
    "io"
    "strings"
    "encoding/json"
    "log"
)

func convert(encoding string, src string) string {
    if encoding == "utf-8" {
        return src
    }
	cd, err := iconv.Open("utf-8", encoding)
	if err != nil {
		fmt.Println("iconv.Open failed!")
		return src
	}
	defer cd.Close()

	target := cd.ConvString(src)
    return target
}

func get(url string, user_agent string) string {
    client := &http.Client{}
    request, err := http.NewRequest("GET", url, nil)
    if err != nil {
        fmt.Println("http.Get failed")
        return url
    }
    request.Header.Add("User-Agent", user_agent)
    resp, err := client.Do(request)
    if err != nil {
        fmt.Println("http.Get failed")
        return url
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        fmt.Println("io.ReadAll failed")
        return url
    }
    return string(body)
}

func parse_meta(src string) map[string]string {
    meta := make(map[string]string)

    doc, err := goquery.NewDocumentFromReader(strings.NewReader(src))
    if err != nil {
        fmt.Println("goquery.NewDocumentFromReader failed")
        return meta
    }
    doc.Find("meta").Each(func(_ int, s *goquery.Selection) {
        name, exist := s.Attr("name")
        if exist {
            meta[strings.ToLower(name)] = s.AttrOr("content", "")
        } else {
            equiv, exist := s.Attr("http-equiv") 
            if exist {
                meta[strings.ToLower(equiv)] = s.AttrOr("content", "")
            } else {
                charset, exist := s.Attr("charset")
                if exist {
                    meta["charset"] = strings.ToLower(charset)
                }
            }
        }
        if meta["charset"] == "" && meta["content-type"] != "" {
            splits := strings.Split(meta["content-type"], "charset=")
            if len(splits) == 2 {
                meta["charset"] = splits[1]
            }
        }
    })
    meta["title"] = strings.TrimSpace(doc.Find("title").First().Text())

    return meta
}

func filter_links(links []string) []string {

    if len(links) < 2 {
        return links
    }

    length := 0
    for _, link := range links {
        length += len(link)
    }
    avg := length / len(links)

    results := []string{}

    for _, link := range links {
        if len(link) >= avg {
            results = append(results, link)
        }
    }

    return results
}

func parse_links(src string) []string {
    links := []string{}
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(src))
    if err != nil {
        fmt.Println("goquery.NewDocumentFromReader failed")
        return links
    }
    doc.Find("a").Each(func(_ int, s * goquery.Selection) {
        text := strings.TrimSpace(s.Text())
        if len(text) > 0 {
            links = append(links, text)
        }
    })
    links = filter_links(links)
    return links
}

func parse(src string) Info {
    meta := parse_meta(src)
    if meta["charset"] != "utf-8" && meta["charset"] != "" {
        src = convert(meta["charset"], src)
        meta = parse_meta(src)
    }
    links := parse_links(src)

    info := Info{meta, links}

    return info
}

type Info struct {
    Meta map[string]string
    Links []string
}

func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("content-type","application/json")
    keys, ok := r.URL.Query()["url"]
    user_agent := strings.Join(r.Header["User-Agent"], "")
    if ok {
        url := keys[0]
        src := get(url, user_agent)
        info := parse(src)
        json_byte, _ := json.Marshal(info)
        json_string := string(json_byte)
        fmt.Fprint(w, json_string)
    } else {
        fmt.Fprintf(w, `{"error": "url required"}`)
    }
}

func main() {
    http.HandleFunc("/", handler)
    fmt.Printf("Starting server at port 8080\n")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
