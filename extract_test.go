package extract

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var extractTests = []struct {
	filename string
	title    string
}{
	{"interests.html", "興味の変遷 - Weblog - Hail2u.net"},
	{"20160709190837", "日報 2016-07-10 - 日報"},
	{"1388078389", "伝説のベイジアン先生にベイズの基礎を教えてもらえる「図解・ベイズ統計「超」入門」を読んだ - EchizenBlog-Zwei"},
	{"20160710180958", "日報 2016-07-10 - 日報"},
}

func TestExtract(t *testing.T) {
	for i, test := range extractTests {
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

func BenchmarkExtract(b *testing.B) {
	for i, test := range extractTests {
		b.Run(fmt.Sprint("L", i), func(b *testing.B) { benchmarkExtract(test.filename, b) })
	}
}

func benchmarkExtract(filename string, b *testing.B) {
	path := filepath.Join("testdata", filename)
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		b.Fatalf("could not read %v: %v", path, err)
	}
	b.SetBytes(int64(len(buf)))
	runtime.GC()
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rd := bytes.NewBuffer(buf)
		_, _, err := Extract(rd)
		if err != nil {
			b.Log(path)
			b.Fatal(err)
		}
	}
}

var indexWordTests = []struct{
	a string
	b string
	i int
}{
	{"foo", "baz", -1},
	{"pager", "page", -1},
	{"pager", "pager", 0},
	{"new-pager", "pager", 4},
	{"pager_left", "pager", 0},
	{"newpagertext foo", "pager", -1},
	{"block-pager page foo", "pager", 6},
}

func TestIndexWord(t *testing.T) {
	for _, test := range indexWordTests {
		actual := indexWord(test.a, test.b)
		if actual != test.i {
			t.Errorf("indexWord(%q,%q) = %v; want %v", test.a, test.b, actual, test.i)
		}
	}
}

func TestFuzzCrashers(t *testing.T) {
	var crashers = []string{
		"<title>",
	}

	for _, f := range crashers {
		Extract(strings.NewReader(f))
	}
}
