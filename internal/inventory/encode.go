package inventory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"
)

func validStrings(v reflect.Value) error {
	if !v.IsValid() {
		return nil
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if !v.IsNil() {
			return validStrings(v.Elem())
		}
	case reflect.String:
		if !utf8.ValidString(v.String()) {
			return fmt.Errorf("invalid UTF-8 string")
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if e := validStrings(v.Field(i)); e != nil {
				return e
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if e := validStrings(v.Index(i)); e != nil {
				return e
			}
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if e := validStrings(iter.Key()); e != nil {
				return e
			}
			if e := validStrings(iter.Value()); e != nil {
				return e
			}
		}
	}
	return nil
}

// Encode validates and copies its input, then emits deterministic UTF-8 JSON with final LF.
func Encode(a Artifact) ([]byte, error) {
	if a == nil || reflect.ValueOf(a).IsNil() {
		return nil, fmt.Errorf("nil artifact")
	}
	if e := validStrings(reflect.ValueOf(a)); e != nil {
		return nil, e
	}
	raw, e := json.Marshal(a)
	if e != nil {
		return nil, e
	}
	copy, e := Decode(raw)
	if e != nil {
		return nil, e
	}
	switch x := copy.(type) {
	case *Snapshot:
		for i, f := range x.Findings {
			n, e := NormalizeFinding(x.TargetID, f)
			if e != nil {
				return nil, e
			}
			x.Findings[i] = n
		}
		slices.SortFunc(x.Findings, func(a, b Finding) int { return strings.Compare(a.FindingID, b.FindingID) })
		for i := range x.Evidence {
			if v, ok := x.Evidence[i].Values.(ProcessValues); ok {
				v.Observed = normalizeFacts(v.Observed).(NetworkFacts)
				x.Evidence[i].Values = v
			}
		}
		slices.SortFunc(x.Evidence, func(a, b Evidence) int { return strings.Compare(a.EvidenceID, b.EvidenceID) })
		for i := range x.Coverage {
			c := &x.Coverage[i]
			c.AttemptedSources = stringsSet(c.AttemptedSources)
			c.DiagnosticCodes = stringsSet(c.DiagnosticCodes)
			c.Limitations = stringsSet(c.Limitations)
		}
		slices.SortFunc(x.Coverage, func(a, b Coverage) int { return strings.Compare(string(a.Collector), string(b.Collector)) })
	case *Assessments:
		for i := range x.Entries {
			a := &x.Entries[i]
			a.Selector.Identity, e = NormalizeIdentity(a.Selector.Kind, a.Selector.Identity)
			if e != nil {
				return nil, e
			}
			if b, ok := a.Basis.(AdvanceBasis); ok {
				normalizeExpected(&b.Expectation.Facts)
				a.Basis = b
			}
		}
		slices.SortFunc(x.Entries, func(a, b Assessment) int { return strings.Compare(a.AssessmentID, b.AssessmentID) })
	case *Review:
		x.AssessmentIDs = stringsSet(x.AssessmentIDs)
		x.UnmatchedAssessmentIDs = stringsSet(x.UnmatchedAssessmentIDs)
		for i := range x.Items {
			x.Items[i].AssessmentIDs = stringsSet(x.Items[i].AssessmentIDs)
			x.Items[i].ReasonCodes = stringsSet(x.Items[i].ReasonCodes)
		}
		slices.SortFunc(x.Items, func(a, b ReviewItem) int { return strings.Compare(a.FindingID, b.FindingID) })
	}
	// Canonical first avoids encoding/json's HTML and U+2028 escaping.
	compact, e := Canonical(copy)
	if e != nil {
		return nil, e
	}
	var out bytes.Buffer
	if e = json.Indent(&out, compact, "", "  "); e != nil {
		return nil, e
	}
	if out.Len()+1 > MaxArtifactBytes {
		return nil, fmt.Errorf("formatted artifact exceeds %d bytes", MaxArtifactBytes)
	}
	return append(out.Bytes(), '\n'), nil
}

func normalizeExpected(f *ExpectedFacts) {
	if f.Owners != nil {
		v := sortedSet(*f.Owners, ownerCompare)
		f.Owners = &v
	}
	if f.RunAsUsers != nil {
		v := stringsSet(*f.RunAsUsers)
		f.RunAsUsers = &v
	}
	if f.RunAsGroups != nil {
		v := stringsSet(*f.RunAsGroups)
		f.RunAsGroups = &v
	}
	if f.LocalMembers != nil {
		v := sortedSet(*f.LocalMembers, memberCompare)
		f.LocalMembers = &v
	}
	if f.Commands != nil {
		v := append([]Command{}, (*f.Commands)...)
		for i := range v {
			v[i].Tags = stringsSet(v[i].Tags)
		}
		f.Commands = &v
	}
	if f.PolicyContext != nil {
		v := append([]string{}, (*f.PolicyContext)...)
		for i := range v {
			v[i] = stripTerminator(v[i])
		}
		f.PolicyContext = &v
	}
	if f.Raw != nil {
		v := stripTerminator(*f.Raw)
		f.Raw = &v
	}
}
