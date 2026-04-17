package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andreigliga/pack-calculator/internal/calculator"
	"github.com/andreigliga/pack-calculator/internal/packsize"
	"github.com/andreigliga/pack-calculator/internal/webui"
)

type stubPackSvc struct {
	sizes   []int
	getErr  error
	replErr error
}

func (s *stubPackSvc) Get(ctx context.Context) ([]int, error) {
	return s.sizes, s.getErr
}

func (s *stubPackSvc) Replace(ctx context.Context, sizes []int) ([]int, error) {
	if s.replErr != nil {
		return nil, s.replErr
	}
	s.sizes = sizes
	return sizes, nil
}

func newRouterWithStub(stub *stubPackSvc) http.Handler {
	h := &Handlers{Packs: stub}
	return NewRouter(h, nil)
}

func TestGetPackSizes_StorageError(t *testing.T) {
	stub := &stubPackSvc{getErr: errors.New("db unavailable")}
	r := newRouterWithStub(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/pack-sizes", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetPackSizes_NilSliceBecomesEmptyJSON(t *testing.T) {
	stub := &stubPackSvc{sizes: nil}
	r := newRouterWithStub(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/pack-sizes", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var got packSizesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Sizes) != 0 {
		t.Fatalf("want empty sizes, got %#v", got.Sizes)
	}
}

func TestGetPackSizes(t *testing.T) {
	stub := &stubPackSvc{sizes: []int{250, 500, 1000}}
	r := newRouterWithStub(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/pack-sizes", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got packSizesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := []int{250, 500, 1000}
	if len(got.Sizes) != len(want) {
		t.Fatalf("got %v want %v", got.Sizes, want)
	}
}

func TestPutPackSizes_Success(t *testing.T) {
	stub := &stubPackSvc{}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`{"sizes":[250,500,1000]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPutPackSizes_ValidationError(t *testing.T) {
	stub := &stubPackSvc{
		replErr: wrap(packsize.ErrInvalidSizes, "must contain at least one size"),
	}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`{"sizes":[]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPutPackSizes_ReplaceServerError(t *testing.T) {
	stub := &stubPackSvc{replErr: errors.New("transaction failed")}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`{"sizes":[1,2,3]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPutPackSizes_UnknownJSONField(t *testing.T) {
	stub := &stubPackSvc{}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`{"sizes":[1],"extra":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPutPackSizes_InvalidJSON(t *testing.T) {
	stub := &stubPackSvc{}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCalculate_GetStoredSizesError(t *testing.T) {
	stub := &stubPackSvc{getErr: errors.New("read failed")}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`{"items":10}`)
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCalculate_InvalidInlinePackSize(t *testing.T) {
	stub := &stubPackSvc{sizes: []int{250}}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`{"items":10,"sizes":[0]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCalculate_InvalidJSON(t *testing.T) {
	stub := &stubPackSvc{}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`{`)
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestCalculate_UnknownJSONField(t *testing.T) {
	stub := &stubPackSvc{sizes: []int{10}}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`{"items":1,"unknown":1}`)
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCalculate_UsesStoredSizes(t *testing.T) {
	stub := &stubPackSvc{sizes: []int{250, 500, 1000, 2000, 5000}}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`{"items":12001}`)
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got calculator.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.TotalItems != 12250 || got.TotalPacks != 4 {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestCalculate_InlineSizes(t *testing.T) {
	stub := &stubPackSvc{sizes: nil}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`{"items":500000,"sizes":[23,31,53]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got calculator.Result
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.TotalItems != 500_000 || got.TotalPacks != 9438 {
		t.Fatalf("edge case mismatch: %+v", got)
	}
}

func TestCalculate_NoSizesAvailable(t *testing.T) {
	stub := &stubPackSvc{sizes: nil}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`{"items":100}`)
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCalculate_NegativeItems(t *testing.T) {
	stub := &stubPackSvc{sizes: []int{250}}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`{"items":-1}`)
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestCalculate_ExceedsMax(t *testing.T) {
	stub := &stubPackSvc{sizes: []int{250}}
	r := newRouterWithStub(stub)

	body := bytes.NewBufferString(`{"items":1000000000}`)
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHealthz(t *testing.T) {
	r := newRouterWithStub(&stubPackSvc{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestCORS_PreflightOPTIONS(t *testing.T) {
	r := newRouterWithStub(&stubPackSvc{})
	req := httptest.NewRequest(http.MethodOptions, "/api/calculate", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing CORS header")
	}
}

func TestStaticUI_ServesIndex(t *testing.T) {
	h := &Handlers{Packs: &stubPackSvc{}}
	r := NewRouter(h, webui.FS())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(strings.ToLower(body), "html") {
		t.Fatalf("expected HTML, got %q", body[:min(80, len(body))])
	}
}

func wrap(sentinel error, msg string) error {
	return &wrappedErr{sentinel: sentinel, msg: msg}
}

type wrappedErr struct {
	sentinel error
	msg      string
}

func (w *wrappedErr) Error() string { return w.sentinel.Error() + ": " + w.msg }
func (w *wrappedErr) Unwrap() error { return w.sentinel }
