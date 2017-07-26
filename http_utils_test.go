package fidias

import (
	"fmt"
	"net/http/httptest"
	"testing"
)

func Test_writeJSONResponse_data(t *testing.T) {
	w := httptest.NewRecorder()
	headers := map[string]string{"test": "value"}
	data := &KeyValueItem{Key: "foo"}
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

func Test_writeJSONResponse_error(t *testing.T) {
	w := httptest.NewRecorder()
	headers := map[string]string{}
	data := &KeyValueItem{Key: "foo"}
	err := fmt.Errorf("foo")
	writeJSONResponse(w, 200, headers, data, err)

	resp := w.Result()
	if resp.StatusCode < 400 {
		t.Fatal("code should be > 400")
	}
}
