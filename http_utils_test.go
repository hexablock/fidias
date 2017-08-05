package fidias

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	chord "github.com/hexablock/go-chord"
)

func Test_parseIntQueryParam(t *testing.T) {
	req, _ := http.NewRequest("GET", "/foo?r=3", nil)
	r, err := parseIntQueryParam(req, "r")
	if err != nil {
		t.Fatal(err)
	}
	if r != 3 {
		t.Fatal("param mismatch")
	}

	req2, _ := http.NewRequest("GET", "/foo", nil)
	r2, _ := parseIntQueryParam(req2, "r")
	if r2 != 0 {
		t.Fatal("r should be 0")
	}
}

func Test_writeJSONResponse_data(t *testing.T) {
	w := httptest.NewRecorder()
	headers := map[string]string{"test": "value"}
	data := &KeyValuePair{Key: []byte("foo")}
	writeJSONResponse(w, 200, headers, data, nil)

	resp := w.Result()

	cth := resp.Header["Content-Type"]
	if cth == nil || len(cth) == 0 || cth[0] != "application/json" {
		t.Fatal("header mismatch")
	}

	tst := resp.Header["Test"]
	if tst == nil || len(tst) == 0 || tst[0] != "value" {
		t.Fatal("header mismatch", tst)
	}

	w2 := httptest.NewRecorder()
	headers = map[string]string{}
	d2 := []byte("foo")
	//err := fmt.Errorf("error foo")
	writeJSONResponse(w, 200, headers, d2, nil)

	rsp2 := w2.Result()
	cth = rsp2.Header["Content-Type"]
	if cth != nil {
		t.Fatal("should be nil")
	}
}

func Test_generateRedirect(t *testing.T) {
	s, err := generateRedirect(&chord.Vnode{Meta: []byte("http=foo")}, "/foo/bar")
	if err != nil {
		t.Fatal(err)
	}
	if s != "http://foo/foo/bar" {
		t.Fatal("url mismatch")
	}
}

func Test_writeJSONResponse_error(t *testing.T) {
	w := httptest.NewRecorder()
	headers := map[string]string{}
	data := &KeyValuePair{Key: []byte("foo")}
	err := fmt.Errorf("foo")
	writeJSONResponse(w, 200, headers, data, err)

	resp := w.Result()
	if resp.StatusCode < 400 {
		t.Fatal("code should be > 400")
	}
}
