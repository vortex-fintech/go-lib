package promreporter

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
)

type obsCall struct {
	service string
	method  string
	code    string
	seconds float64
}

type errCall struct {
	typ     string
	service string
	method  string
}

type fakeMetrics struct {
	obs  []obsCall
	errs []errCall
}

func (m *fakeMetrics) ObserveRPC(service, method, code string, seconds float64) {
	m.obs = append(m.obs, obsCall{service, method, code, seconds})
}
func (m *fakeMetrics) IncError(typ, service, method string) {
	m.errs = append(m.errs, errCall{typ, service, method})
}

func TestReporter_ObserveRPCFull_OK(t *testing.T) {
	t.Parallel()

	fm := &fakeMetrics{}
	r := Reporter{M: fm}

	r.ObserveRPCFull(context.Background(), "/pkg.Service/Method", codes.OK, 0.123)

	if len(fm.obs) != 1 {
		t.Fatalf("ObserveRPC not called, got %d", len(fm.obs))
	}
	got := fm.obs[0]
	if got.service != "pkg.Service" || got.method != "Method" || got.code != "OK" {
		t.Fatalf("labels mismatch: %+v", got)
	}
	if len(fm.errs) != 0 {
		t.Fatalf("IncError should not be called on OK, got %d", len(fm.errs))
	}
}

func TestReporter_ObserveRPCFull_Error(t *testing.T) {
	t.Parallel()

	fm := &fakeMetrics{}
	r := Reporter{M: fm}

	r.ObserveRPCFull(context.Background(), "/pkg.Service/Login", codes.PermissionDenied, 0.050)

	if len(fm.obs) != 1 {
		t.Fatalf("ObserveRPC not called, got %d", len(fm.obs))
	}
	if len(fm.errs) != 1 {
		t.Fatalf("IncError expected=1 got=%d", len(fm.errs))
	}

	obs := fm.obs[0]
	if obs.service != "pkg.Service" || obs.method != "Login" || obs.code != "PermissionDenied" {
		t.Fatalf("labels mismatch: %+v", obs)
	}
	errc := fm.errs[0]
	if errc.typ != "grpc" || errc.service != "pkg.Service" || errc.method != "Login" {
		t.Fatalf("error labels mismatch: %+v", errc)
	}
}

func TestReporter_NoMetrics_NoPanic(t *testing.T) {
	t.Parallel()

	var r Reporter
	r.ObserveRPCFull(context.Background(), "/s/m", codes.Internal, 0.001)
}

func TestSplitGRPCMethod(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in      string
		wantSvc string
		wantMth string
	}{
		{"/pkg.Service/Method", "pkg.Service", "Method"},
		{"pkg.Service/Method", "pkg.Service", "Method"},
		{"pkg.Service.Method", "pkg.Service", "Method"},
		{"Method", "unknown", "Method"},
		{"", "unknown", "unknown"},
	}

	for _, c := range cases {
		svc, mth := SplitGRPCMethod(c.in)
		if svc != c.wantSvc || mth != c.wantMth {
			t.Fatalf("in=%q got (%q,%q) want (%q,%q)", c.in, svc, mth, c.wantSvc, c.wantMth)
		}
	}
}
