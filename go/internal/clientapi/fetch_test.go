package clientapi

import "testing"

func TestStatusURLAndBody(t *testing.T) {
	if StatusURL("100.64.0.1") != "http://100.64.0.1:18020/status" {
		t.Fatal(StatusURL("100.64.0.1"))
	}
	if StatusURL("fd7a:115c:a1e0::1") != "http://[fd7a:115c:a1e0::1]:18020/status" {
		t.Fatal(StatusURL("fd7a:115c:a1e0::1"))
	}
	if _, ok := ReadStatus(200, []byte(`{"hostname":"peer"}`)); !ok {
		t.Fatal("object")
	}
	if _, ok := ReadStatus(200, []byte(`[1]`)); ok {
		t.Fatal("array")
	}
	if _, ok := ReadStatus(404, []byte(`{"hostname":"peer"}`)); ok {
		t.Fatal("status")
	}
}
