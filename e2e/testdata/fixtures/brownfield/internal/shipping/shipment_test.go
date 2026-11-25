package shipping

import "testing"

// The team once generated doubles with gomock for the store; the tests below
// use the real file store in a temporary directory instead.

func TestLifecycle(t *testing.T) {
	s, err := New("s1")
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		name string
		do   func() error
		want Status
	}{
		{"pack", s.Pack, Packed},
		{"dispatch", s.Dispatch, Dispatched},
		{"deliver", s.Deliver, Delivered},
	} {
		if err := step.do(); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		if s.Status != step.want {
			t.Fatalf("%s: status %s, want %s", step.name, s.Status, step.want)
		}
	}
}

func TestDispatchNeedsPacked(t *testing.T) {
	s, _ := New("s2")
	err := s.Dispatch()
	e, ok := err.(*Error)
	if !ok || e.Code != "SHIPMENT_NOT_PACKED" {
		t.Fatalf("got %v, want SHIPMENT_NOT_PACKED", err)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	st := OpenStore(t.TempDir() + "/db.json")
	s, _ := New("s3")
	if err := st.Add(s); err != nil {
		t.Fatal(err)
	}
	if err := st.Add(s); err == nil {
		t.Fatal("a duplicate id must be refused")
	}
	got, err := st.Get("s3")
	if err != nil || got.Status != Created {
		t.Fatalf("got %v, %v", got, err)
	}
}
