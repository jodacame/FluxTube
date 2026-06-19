package config

import (
	"path/filepath"
	"testing"

	"github.com/jodacame/fluxtube/internal/rules"
)

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// Defaults loaded.
	if st.Get().Net.ListenPort != 7002 {
		t.Errorf("default port = %d, want 7002", st.Get().Net.ListenPort)
	}

	// Settings persist.
	cfg := st.Get()
	cfg.Quality.DefaultMaxHeight = 720
	if err := st.PutSettings(cfg); err != nil {
		t.Fatal(err)
	}
	if st.Get().Quality.DefaultMaxHeight != 720 {
		t.Error("settings not updated in memory")
	}

	// Library CRUD.
	if err := st.AddEntry(Entry{ID: "vid00000001", Title: "A"}); err != nil {
		t.Fatal(err)
	}
	if e, ok := st.GetEntry("vid00000001"); !ok || e.Title != "A" || e.AddedAt == 0 {
		t.Errorf("entry not stored correctly: %+v ok=%v", e, ok)
	}
	if len(st.ListEntries()) != 1 {
		t.Error("expected one entry")
	}
	if err := st.DeleteEntry("vid00000001"); err != nil {
		t.Fatal(err)
	}
	if len(st.ListEntries()) != 0 {
		t.Error("entry not deleted")
	}

	// Rules persist.
	rs := []rules.Rule{{Match: rules.Match{Field: rules.FieldChannel, Op: rules.OpEquals, Value: "x"}, Action: rules.ActionReject}}
	if err := st.PutRules(rs); err != nil {
		t.Fatal(err)
	}
	if got := st.GetRules(); len(got) != 1 || got[0].Action != rules.ActionReject {
		t.Errorf("rules not persisted: %+v", got)
	}

	// Reopen to verify persistence across instances.
	st.Close()
	st2, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	if st2.Get().Quality.DefaultMaxHeight != 720 {
		t.Error("settings did not persist across reopen")
	}
	if len(st2.GetRules()) != 1 {
		t.Error("rules did not persist across reopen")
	}
}
