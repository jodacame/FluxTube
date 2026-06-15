// Package rules evaluates per-source rules against a video, mirroring the
// FluxTorrent rules model but matching on YouTube channel, title or video id.
package rules

import (
	"regexp"
	"strings"
)

// Field is the video attribute a rule matches against.
type Field string

const (
	FieldChannel Field = "channel"
	FieldTitle   Field = "title"
	FieldVideoID Field = "videoId"
)

// Op is the comparison operator.
type Op string

const (
	OpEquals   Op = "equals"
	OpContains Op = "contains"
	OpRegex    Op = "regex"
)

// Action is what a matching rule applies.
type Action string

const (
	ActionReject          Action = "reject"
	ActionMaxQuality      Action = "maxQuality"
	ActionPreferAudioLang Action = "preferAudioLang"
	ActionPreferSubLang   Action = "preferSubLang"
	ActionCache           Action = "cache"
	ActionEphemeral       Action = "ephemeral"
)

// Match describes the condition for a rule.
type Match struct {
	Field Field  `json:"field"`
	Op    Op     `json:"op"`
	Value string `json:"value"`
}

// Rule is a single ordered rule.
type Rule struct {
	Match     Match  `json:"match"`
	Action    Action `json:"action"`
	MaxHeight int    `json:"maxHeight,omitempty"` // for maxQuality
	Lang      string `json:"lang,omitempty"`      // for preferAudioLang / preferSubLang
	Note      string `json:"note,omitempty"`
}

// Subject is the set of video attributes evaluated against rules.
type Subject struct {
	VideoID string
	Title   string
	Channel string
}

// Decision is the resolved outcome after evaluating all rules in order.
type Decision struct {
	Reject          bool
	RejectReason    string
	MaxHeight       int    // 0 = no cap from rules
	PreferAudioLang string // empty = none
	PreferSubLang   string // empty = none
	ForceCache      bool
	ForceEphemeral  bool
}

// Evaluate applies rules in order (first match per concern wins) and returns
// the combined decision.
func Evaluate(list []Rule, s Subject) Decision {
	var d Decision
	for _, r := range list {
		if !matches(r.Match, s) {
			continue
		}
		switch r.Action {
		case ActionReject:
			if !d.Reject {
				d.Reject = true
				d.RejectReason = firstNonEmpty(r.Note, "rejected by rule")
			}
		case ActionMaxQuality:
			if d.MaxHeight == 0 && r.MaxHeight > 0 {
				d.MaxHeight = r.MaxHeight
			}
		case ActionPreferAudioLang:
			if d.PreferAudioLang == "" {
				d.PreferAudioLang = r.Lang
			}
		case ActionPreferSubLang:
			if d.PreferSubLang == "" {
				d.PreferSubLang = r.Lang
			}
		case ActionCache:
			if !d.ForceEphemeral {
				d.ForceCache = true
			}
		case ActionEphemeral:
			if !d.ForceCache {
				d.ForceEphemeral = true
			}
		}
	}
	return d
}

func matches(m Match, s Subject) bool {
	var target string
	switch m.Field {
	case FieldChannel:
		target = s.Channel
	case FieldTitle:
		target = s.Title
	case FieldVideoID:
		target = s.VideoID
	default:
		return false
	}
	switch m.Op {
	case OpEquals:
		return strings.EqualFold(target, m.Value)
	case OpContains:
		return strings.Contains(strings.ToLower(target), strings.ToLower(m.Value))
	case OpRegex:
		re, err := regexp.Compile(m.Value)
		if err != nil {
			return false
		}
		return re.MatchString(target)
	}
	return false
}

// Valid reports whether a rule is well-formed.
func (r Rule) Valid() bool {
	switch r.Match.Field {
	case FieldChannel, FieldTitle, FieldVideoID:
	default:
		return false
	}
	switch r.Match.Op {
	case OpEquals, OpContains, OpRegex:
	default:
		return false
	}
	if r.Match.Op == OpRegex {
		if _, err := regexp.Compile(r.Match.Value); err != nil {
			return false
		}
	}
	switch r.Action {
	case ActionReject, ActionMaxQuality, ActionPreferAudioLang, ActionPreferSubLang, ActionCache, ActionEphemeral:
		return true
	}
	return false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
