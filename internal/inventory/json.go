package inventory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Limits bound resource use with explicit errors; no input is truncated.
const (
	MaxArtifactBytes       = 16 << 20
	MaxStringBytes         = 256 << 10
	MaxEntries             = 100000
	MaxDepth               = 32
	MaxNodes               = 1000000
	MaxInteger       int64 = 9007199254740991
)

// Decode rejects ambiguous JSON before constructing typed domain values.
func Decode(data []byte) (Artifact, error) {
	v, err := parseJSON(data)
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("artifact must be an object")
	}
	var a Artifact
	switch m["kind"] {
	case "snapshot":
		a = &Snapshot{}
	case "assessments":
		a = &Assessments{}
	case "review":
		a = &Review{}
	default:
		return nil, fmt.Errorf("unsupported artifact kind %v", m["kind"])
	}
	dst := reflect.ValueOf(a).Elem()
	if err := assign(dst, v, m, "$", false); err != nil {
		return nil, err
	}
	if err := validate(a); err != nil {
		return nil, err
	}
	return a, nil
}

func parseJSON(data []byte) (any, error) {
	if len(data) > MaxArtifactBytes {
		return nil, fmt.Errorf("artifact exceeds %d bytes", MaxArtifactBytes)
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("invalid UTF-8")
	}
	// encoding/json replaces lone surrogates; reject them before decoding.
	for i := 0; i < len(data); i++ {
		if data[i] != '\\' {
			continue
		}
		i++
		if i >= len(data) {
			break
		}
		if data[i] != 'u' {
			continue
		}
		if i+4 >= len(data) {
			break
		}
		n, e := strconv.ParseUint(string(data[i+1:i+5]), 16, 16)
		if e != nil {
			continue
		}
		i += 4
		if n >= 0xdc00 && n <= 0xdfff {
			return nil, fmt.Errorf("unpaired Unicode surrogate")
		}
		if n >= 0xd800 && n <= 0xdbff {
			if i+6 >= len(data) || string(data[i+1:i+3]) != "\\u" {
				return nil, fmt.Errorf("unpaired Unicode surrogate")
			}
			low, e := strconv.ParseUint(string(data[i+3:i+7]), 16, 16)
			if e != nil || low < 0xdc00 || low > 0xdfff {
				return nil, fmt.Errorf("unpaired Unicode surrogate")
			}
			i += 6
		}
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	nodes := 0
	v, err := readValue(d, 0, &nodes)
	if err != nil {
		return nil, err
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing JSON data: %v", err)
	}
	return v, nil
}
func readValue(d *json.Decoder, depth int, nodes *int) (any, error) {
	*nodes++
	if depth > MaxDepth || *nodes > MaxNodes {
		return nil, fmt.Errorf("JSON depth/node limit exceeded")
	}
	t, err := d.Token()
	if err != nil {
		return nil, err
	}
	switch x := t.(type) {
	case json.Delim:
		switch x {
		case '{':
			m := map[string]any{}
			for d.More() {
				key, e := d.Token()
				if e != nil {
					return nil, e
				}
				k, ok := key.(string)
				if !ok {
					return nil, fmt.Errorf("invalid key")
				}
				if len(k) > MaxStringBytes {
					return nil, fmt.Errorf("key size limit exceeded")
				}
				if _, ok = m[k]; ok {
					return nil, fmt.Errorf("duplicate key %q", k)
				}
				if len(m) >= MaxEntries {
					return nil, fmt.Errorf("object count limit exceeded")
				}
				v, e := readValue(d, depth+1, nodes)
				if e != nil {
					return nil, e
				}
				m[k] = v
			}
			_, err = d.Token()
			return m, err
		case '[':
			a := []any{}
			for d.More() {
				if len(a) >= MaxEntries {
					return nil, fmt.Errorf("array count limit exceeded")
				}
				v, e := readValue(d, depth+1, nodes)
				if e != nil {
					return nil, e
				}
				a = append(a, v)
			}
			_, err = d.Token()
			return a, err
		}
	case string:
		if len(x) > MaxStringBytes {
			return nil, fmt.Errorf("string exceeds %d bytes", MaxStringBytes)
		}
	case json.Number:
		n, e := strconv.ParseInt(string(x), 10, 64)
		if e != nil || n > MaxInteger || n < -MaxInteger {
			return nil, fmt.Errorf("invalid exact integer %q", x)
		}
	}
	return t, nil
}

type fieldInfo struct {
	index    []int
	optional bool
}

func fields(t reflect.Type, prefix []int, out map[string]fieldInfo) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		idx := append(append([]int{}, prefix...), i)
		if f.Anonymous {
			fields(f.Type, idx, out)
			continue
		}
		tag := strings.Split(f.Tag.Get("json"), ",")
		out[tag[0]] = fieldInfo{idx, len(tag) > 1 && tag[1] == "omitempty"}
	}
}
func assign(dst reflect.Value, src any, context map[string]any, path string, nullable bool) error {
	fail := func() error { return fmt.Errorf("%s: invalid %s", path, dst.Type()) }
	if src == nil {
		if nullable {
			return nil
		}
		return fmt.Errorf("%s: null is forbidden", path)
	}
	if dst.Kind() == reflect.Pointer {
		dst.Set(reflect.New(dst.Type().Elem()))
		return assign(dst.Elem(), src, context, path, false)
	}
	switch dst.Kind() {
	case reflect.Interface:
		var concrete any
		kind, _ := context["kind"].(string)
		switch dst.Type() {
		case reflect.TypeFor[Identity]():
			switch Kind(kind) {
			case USBDevice:
				concrete = USBIdentity{}
			case ExternalStorage:
				concrete = StorageIdentity{}
			case Account:
				concrete = AccountIdentity{}
			case Endpoint:
				concrete = EndpointIdentity{}
			case Rule, Unparsed, Forwarding:
				concrete = SourceIdentity{}
			}
		case reflect.TypeFor[Facts]():
			switch Kind(kind) {
			case USBDevice:
				concrete = USBFacts{}
			case ExternalStorage:
				concrete = StorageFacts{}
			case Account:
				concrete = AccountFacts{}
			case Endpoint:
				concrete = NetworkFacts{}
			case Forwarding:
				concrete = ForwardingFacts{}
			case Rule:
				concrete = SudoFacts{}
			case Unparsed:
				concrete = UnparsedFacts{}
			}
		case reflect.TypeFor[Basis]():
			switch context["origin"] {
			case "reviewed_snapshot":
				concrete = ReviewedBasis{}
			case "advance":
				concrete = AdvanceBasis{}
			}
		case reflect.TypeFor[EvidenceValues]():
			switch context["source_kind"] {
			case "usb_metadata":
				concrete = USBValues{}
			case "local_identity":
				concrete = AccountValues{}
			case "proc":
				concrete = ProcessValues{}
			case "local_file", "command_output":
				concrete = FileValues{}
			}
		}
		if concrete == nil {
			return fmt.Errorf("%s: unsupported payload discriminator", path)
		}
		v := reflect.New(reflect.TypeOf(concrete)).Elem()
		if err := assign(v, src, context, path, false); err != nil {
			return err
		}
		dst.Set(v)
	case reflect.Struct:
		m, ok := src.(map[string]any)
		if !ok {
			return fail()
		}
		fs := map[string]fieldInfo{}
		fields(dst.Type(), nil, fs)
		ctx := make(map[string]any, len(context)+len(m))
		for k, v := range context {
			ctx[k] = v
		}
		for k, v := range m {
			ctx[k] = v
		}
		if selector, ok := m["selector"].(map[string]any); ok {
			ctx["kind"] = selector["kind"]
		}
		if dst.Type() == reflect.TypeFor[ExpectedFacts]() {
			var t reflect.Type
			switch ctx["kind"] {
			case string(Endpoint):
				t = reflect.TypeFor[NetworkFacts]()
			case string(Forwarding):
				t = reflect.TypeFor[ForwardingFacts]()
			case string(Rule):
				t = reflect.TypeFor[SudoFacts]()
			case string(Unparsed):
				t = reflect.TypeFor[UnparsedFacts]()
			}
			if t == nil {
				return fail()
			}
			allowed := map[string]fieldInfo{}
			fields(t, nil, allowed)
			for k := range fs {
				if _, ok := allowed[k]; !ok {
					delete(fs, k)
				}
			}
		}
		for k, v := range m {
			f, ok := fs[k]
			if !ok {
				return fmt.Errorf("%s: unknown field %q", path, k)
			}
			if err := assign(dst.FieldByIndex(f.index), v, ctx, path+"."+k, k == "previous_snapshot_id"); err != nil {
				return err
			}
		}
		for k, f := range fs {
			if _, ok := m[k]; !ok && !f.optional {
				return fmt.Errorf("%s: missing field %q", path, k)
			}
		}
	case reflect.Map:
		m, ok := src.(map[string]any)
		if !ok {
			return fail()
		}
		dst.Set(reflect.MakeMap(dst.Type()))
		for k, v := range m {
			key := reflect.New(dst.Type().Key()).Elem()
			key.SetString(k)
			val := reflect.New(dst.Type().Elem()).Elem()
			if err := assign(val, v, context, path+"."+k, false); err != nil {
				return err
			}
			dst.SetMapIndex(key, val)
		}
	case reflect.Slice:
		a, ok := src.([]any)
		if !ok {
			return fail()
		}
		dst.Set(reflect.MakeSlice(dst.Type(), len(a), len(a)))
		for i, v := range a {
			if err := assign(dst.Index(i), v, context, fmt.Sprintf("%s[%d]", path, i), false); err != nil {
				return err
			}
		}
	case reflect.String:
		s, ok := src.(string)
		if !ok {
			return fail()
		}
		dst.SetString(s)
	case reflect.Bool:
		b, ok := src.(bool)
		if !ok {
			return fail()
		}
		dst.SetBool(b)
	case reflect.Int, reflect.Int64:
		n, ok := src.(json.Number)
		if !ok {
			return fail()
		}
		v, e := strconv.ParseInt(string(n), 10, dst.Type().Bits())
		if e != nil {
			return fail()
		}
		dst.SetInt(v)
	case reflect.Uint16, reflect.Uint32:
		n, ok := src.(json.Number)
		if !ok {
			return fail()
		}
		v, e := strconv.ParseUint(string(n), 10, dst.Type().Bits())
		if e != nil {
			return fail()
		}
		dst.SetUint(v)
	default:
		return fail()
	}
	return nil
}

// ValidateJSON checks bounded JSON structure, duplicate keys and UTF-8 without decoding an artifact.
func ValidateJSON(data []byte) error {
	_, err := parseJSON(data)
	return err
}
