package extract

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestExtract(t *testing.T) {
	var tests = []struct {
		filename string
		title    string
	}{
		{"interests.html", "興味の変遷 - Weblog - Hail2u.net"},
		{"20160709190837", "日報 2016-07-10 - 日報"},
		{"1388078389", "伝説のベイジアン先生にベイズの基礎を教えてもらえる「図解・ベイズ統計「超」入門」を読んだ - EchizenBlog-Zwei"},
		{"20160710180958", "日報 2016-07-10 - 日報"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprint("L", i), func(t *testing.T) { testExtract(test.filename, test.title, t) })
	}
}

func testExtract(filename, expectedTitle string, t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", filename)
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read %v: %v", path, err)
	}
	rd := bytes.NewReader(buf)

	title, content, err := Extract(rd)
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
