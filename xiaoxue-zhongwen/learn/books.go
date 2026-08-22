package learn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	httpClient        = &http.Client{Timeout: 8 * time.Second}
	openLibrarySearch = "https://openlibrary.org/search.json"
	openLibraryCover  = "https://covers.openlibrary.org/b/id/%d-M.jpg"
	openLibraryWork   = "https://openlibrary.org"
)

// SearchBooks looks up children's books related to an interest.
// Lesson text is never taken from these books; they are parent recommendations.
func SearchBooks(query string) BookSearchResult {
	q := strings.TrimSpace(query)
	if q == "" {
		q = "Cristiano Ronaldo children biography"
	}

	result := BookSearchResult{
		Query:  q,
		NoteJA: "本の本文はコピーしないよ。ここに出るのは、保護者向けの関連ブック。学習用の中国語は、かんたんなオリジナル文だよ。",
		NoteZH: "不会照抄书里的句子。这里只给家长找相关童书。孩子读的中文，都是我们写的短句。",
	}

	curated := curatedBooks(q)
	live, err := searchOpenLibrary(q)
	if err != nil {
		result.Offline = true
		result.Books = curated
		return result
	}

	result.Books = mergeBooks(curated, live)
	if len(result.Books) == 0 {
		result.Books = curated
	}
	return result
}

func curatedBooks(query string) []Book {
	all := []Book{
		{
			Title:   "Ronaldo (Ultimate Football Heroes)",
			Authors: "Matt Oldfield, Tom Oldfield",
			Year:    2017,
			InfoURL: "https://openlibrary.org/search?q=Ultimate+Football+Heroes+Ronaldo",
			Source:  "curated",
			NoteJA:  "子ども向けのCロ伝記。島の子どもが選手になる話。",
			NoteZH:  "写给孩子的C罗成长故事：从小岛走到球场。",
			Curated: true,
		},
		{
			Title:   "Cristiano Ronaldo",
			Authors: "children's biographies",
			InfoURL: "https://openlibrary.org/search?q=Cristiano+Ronaldo+children",
			Source:  "curated",
			NoteJA:  "Cロナウドの子どものころを扱った本を探す手がかり。",
			NoteZH:  "用来查找C罗童年主题的儿童传记。",
			Curated: true,
		},
		{
			Title:   "Football picture books for children",
			Authors: "various",
			InfoURL: "https://openlibrary.org/search?q=football+picture+book+children",
			Source:  "curated",
			NoteJA:  "サッカーがすきな子向けの絵本。",
			NoteZH:  "给喜欢足球的孩子看的绘本。",
			Curated: true,
		},
	}
	n := normalizeQuery(query)
	if containsAny(n, ronaldoKeys...) || containsAny(n, footballKeys...) {
		return all
	}
	if containsAny(n, bookKeys...) {
		return []Book{all[2], all[0]}
	}
	return []Book{all[2]}
}

type openLibraryResponse struct {
	Docs []struct {
		Key              string   `json:"key"`
		Title            string   `json:"title"`
		AuthorName       []string `json:"author_name"`
		FirstPublishYear int      `json:"first_publish_year"`
		CoverI           int      `json:"cover_i"`
	} `json:"docs"`
}

func searchOpenLibrary(query string) ([]Book, error) {
	u, err := url.Parse(openLibrarySearch)
	if err != nil {
		return nil, err
	}
	params := u.Query()
	params.Set("q", query)
	params.Set("limit", "8")
	params.Set("fields", "key,title,author_name,first_publish_year,cover_i")
	u.RawQuery = params.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "TankLearnChinese/1.0 (educational; local app)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open library status %d", resp.StatusCode)
	}

	var parsed openLibraryResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	var books []Book
	for _, doc := range parsed.Docs {
		if strings.TrimSpace(doc.Title) == "" {
			continue
		}
		authors := strings.Join(doc.AuthorName, ", ")
		book := Book{
			Title:   doc.Title,
			Authors: authors,
			Year:    doc.FirstPublishYear,
			InfoURL: openLibraryWork + doc.Key,
			Source:  "openlibrary",
			NoteJA:  "Open Library で見つかった関連本。学習文はオリジナルだよ。",
			NoteZH:  "在 Open Library 找到的相关书。课文仍用我们写的短句。",
		}
		if doc.CoverI > 0 {
			book.CoverURL = fmt.Sprintf(openLibraryCover, doc.CoverI)
		}
		books = append(books, book)
	}
	return books, nil
}

func mergeBooks(curated, live []Book) []Book {
	seen := map[string]bool{}
	var out []Book
	add := func(book Book) {
		key := strings.ToLower(strings.TrimSpace(book.Title))
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, book)
	}
	for _, book := range curated {
		add(book)
	}
	for _, book := range live {
		add(book)
	}
	if len(out) > 10 {
		out = out[:10]
	}
	return out
}
