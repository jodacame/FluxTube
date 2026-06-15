package rules

import "testing"

func TestEvaluateFirstMatchWins(t *testing.T) {
	list := []Rule{
		{Match: Match{Field: FieldChannel, Op: OpContains, Value: "news"}, Action: ActionMaxQuality, MaxHeight: 720},
		{Match: Match{Field: FieldChannel, Op: OpContains, Value: "news"}, Action: ActionMaxQuality, MaxHeight: 480},
		{Match: Match{Field: FieldTitle, Op: OpRegex, Value: "(?i)trailer"}, Action: ActionPreferSubLang, Lang: "es"},
		{Match: Match{Field: FieldVideoID, Op: OpEquals, Value: "BANNED123456"}, Action: ActionReject, Note: "blocked"},
	}

	d := Evaluate(list, Subject{Channel: "Daily News", Title: "Movie Trailer"})
	if d.MaxHeight != 720 {
		t.Errorf("MaxHeight = %d, want 720 (first match wins)", d.MaxHeight)
	}
	if d.PreferSubLang != "es" {
		t.Errorf("PreferSubLang = %q, want es", d.PreferSubLang)
	}
	if d.Reject {
		t.Error("should not be rejected")
	}

	d2 := Evaluate(list, Subject{VideoID: "BANNED123456"})
	if !d2.Reject || d2.RejectReason != "blocked" {
		t.Errorf("expected reject with reason, got %+v", d2)
	}
}

func TestCacheEphemeralExclusive(t *testing.T) {
	list := []Rule{
		{Match: Match{Field: FieldChannel, Op: OpEquals, Value: "x"}, Action: ActionCache},
		{Match: Match{Field: FieldChannel, Op: OpEquals, Value: "x"}, Action: ActionEphemeral},
	}
	d := Evaluate(list, Subject{Channel: "x"})
	if !d.ForceCache || d.ForceEphemeral {
		t.Errorf("first cache action should win: %+v", d)
	}
}

func TestValid(t *testing.T) {
	good := Rule{Match: Match{Field: FieldTitle, Op: OpRegex, Value: ".*"}, Action: ActionReject}
	if !good.Valid() {
		t.Error("expected valid rule")
	}
	bad := Rule{Match: Match{Field: FieldTitle, Op: OpRegex, Value: "([unclosed"}, Action: ActionReject}
	if bad.Valid() {
		t.Error("invalid regex should fail validation")
	}
	badAction := Rule{Match: Match{Field: FieldTitle, Op: OpEquals, Value: "x"}, Action: Action("nope")}
	if badAction.Valid() {
		t.Error("unknown action should fail validation")
	}
}
