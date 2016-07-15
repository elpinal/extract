package extract

import (
	"fmt"
	"testing"
)

func TestExtract(t *testing.T) {
	var tests = []struct {
		url   string
		title string
	}{
		// FIXME: use httptest.
		{"http://hail2u.net/blog/internet/interests.html", "興味の変遷 - Weblog - Hail2u.net"},
		{"https://nippo.wikihub.io/@r7kamura/20160709190837", "日報 2016-07-10 - 日報"},
		{"http://d.hatena.ne.jp/echizen_tm/20131226/1388078389", "伝説のベイジアン先生にベイズの基礎を教えてもらえる「図解・ベイズ統計「超」入門」を読んだ - EchizenBlog-Zwei"},
		{"https://nippo.wikihub.io/@woxtu/20160710180958", "日報 2016-07-10 - 日報"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprint("L", i), func(t *testing.T) { testExtract(test.url, test.title, t) })
	}
}

func testExtract(url, expectedTitle string, t *testing.T) {
	t.Parallel()
	title, content, err := ExtractFromURL(url)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) == 0 {
		t.Fatal(`Got no content, expected some content`)
	}
	if title != expectedTitle {
		t.Fatalf("Got %q, want %q", title, expectedTitle)
	}
}
