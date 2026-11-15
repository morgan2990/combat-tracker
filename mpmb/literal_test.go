package mpmb

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseLiteral_ReadsMPMBStateLiterals(t *testing.T) {
	src := `({warlock:{bonus:{'eldritch invocationspact of the chain':[{name:"Pact of the Chain", spells:["find familiar"], times:1, firstCol:"atwill"}]}, shortname:"Warlock", level:1, name:"Warlock (Archfey Patron)", typeSp:"known", refType:"class", factor:[1, "warlock"], known:{cantrips:[2, 2, 3], prepared:false}, calcSpellScores:{dc:14, attack:6}}, 'magic initiate (druid)':{refType:"feat"}})`

	v, err := ParseLiteral(src)
	if err != nil {
		t.Fatalf("ParseLiteral() error = %v", err)
	}

	if got := litString(lit(v, "warlock", "name")); got != "Warlock (Archfey Patron)" {
		t.Errorf("name = %q", got)
	}
	if got, _ := litInt(lit(v, "warlock", "level")); got != 1 {
		t.Errorf("level = %d", got)
	}
	if got := lit(v, "warlock", "factor"); !reflect.DeepEqual(got, []any{1.0, "warlock"}) {
		t.Errorf("factor = %#v", got)
	}
	if got := lit(v, "warlock", "known", "prepared"); got != false {
		t.Errorf("prepared = %#v", got)
	}
	if got := litString(lit(v, "magic initiate (druid)", "refType")); got != "feat" {
		t.Errorf("quoted key lookup = %q", got)
	}
	if got := litString(lit(v, "warlock", "bonus", "eldritch invocationspact of the chain")); got != "" {
		t.Errorf("an array is not a string: %q", got)
	}
}

func TestParseLiteral_Scalars(t *testing.T) {
	for src, want := range map[string]any{
		`({a:1})`:                       map[string]any{"a": 1.0},
		`{a:-2.5, b:'it\'s', c:"q\"x"}`: map[string]any{"a": -2.5, "b": "it's", "c": `q"x`},
		`[1, 2, 3,]`:                    []any{1.0, 2.0, 3.0},
		`({a:true, b:false, c:null})`:   map[string]any{"a": true, "b": false, "c": nil},
		`({"quoted key":1, 'x y':2})`:   map[string]any{"quoted key": 1.0, "x y": 2.0},
		`({s:"a\nbé"})`:                 map[string]any{"s": "a\nbé"},
		// \u and \x escapes, written with an interpreted string so the backslashes stay literal.
		"({s:\"x\\u00e9\\x41\"})": map[string]any{"s": "xéA"},
		`({}) `:                   map[string]any{},
		`({a:[], b:{}})`:          map[string]any{"a": []any{}, "b": map[string]any{}},
		`({a:1e3, b:undefined})`:  map[string]any{"a": 1000.0, "b": nil},
		`  ( { a : 1 , } )  `:     map[string]any{"a": 1.0},
		`({12:"x"})`:              map[string]any{"12": "x"},
	} {
		got, err := ParseLiteral(src)
		if err != nil {
			t.Errorf("ParseLiteral(%q) error = %v", src, err)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ParseLiteral(%q) = %#v, want %#v", src, got, want)
		}
	}
}

func TestParseLiteral_RejectsAnythingThatIsNotPlainData(t *testing.T) {
	for name, src := range map[string]string{
		"function value":  `({atkAdd:{'x':(function (fields, v) { return 1; })}})`,
		"arrow function":  `({a:() => 1})`,
		"identifier":      `({a:someVariable})`,
		"call":            `({a:alert(1)})`,
		"operator":        `({a:1+2})`,
		"unterminated":    `({a:"abc})`,
		"missing colon":   `({a 1})`,
		"missing comma":   `({a:1 b:2})`,
		"unclosed object": `({a:1`,
		"unclosed paren":  `({a:1}`,
		"trailing text":   `({a:1}) extra`,
		"empty":           ``,
		"bad number":      `({a:1.2.3})`,
		"newline in str":  "({a:'x\ny'})",
		"bad escape":      `({a:"\u12"})`,
	} {
		if _, err := ParseLiteral(src); err == nil {
			t.Errorf("%s: ParseLiteral(%q) succeeded, want an error", name, src)
		}
	}
}

func TestParseLiteral_AcceptsVoidZeroAndGroupingButNotOtherOperators(t *testing.T) {
	got, err := ParseLiteral(`({calcSpellScores:(void 0), b:((1)), c:[(void 0)]})`)
	if err != nil {
		t.Fatalf("ParseLiteral() error = %v", err)
	}
	want := map[string]any{"calcSpellScores": nil, "b": 1.0, "c": []any{nil}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
	for _, bad := range []string{`({a:(void 1)})`, `({a:void foo})`, `({a:(1 + 2)})`, `({a:(1})`} {
		if _, err := ParseLiteral(bad); err == nil {
			t.Errorf("ParseLiteral(%q) succeeded, want an error", bad)
		}
	}
}

func TestParseLiteral_BoundsNesting(t *testing.T) {
	deep := strings.Repeat("[", 200) + strings.Repeat("]", 200)
	if _, err := ParseLiteral(deep); err == nil {
		t.Error("deeply nested input was accepted")
	}
	ok := strings.Repeat("[", 20) + strings.Repeat("]", 20)
	if _, err := ParseLiteral(ok); err != nil {
		t.Errorf("moderately nested input rejected: %v", err)
	}
}

func TestLit_MissingPathsAreNil(t *testing.T) {
	v, _ := ParseLiteral(`({a:{b:1}})`)
	if lit(v, "a", "x") != nil || lit(v, "z", "b") != nil || lit(nil, "a") != nil {
		t.Error("a missing path did not return nil")
	}
	if _, ok := litInt("x"); ok {
		t.Error("litInt accepted a string")
	}
}

func TestTitleCase(t *testing.T) {
	if got := titleCase("magic initiate (druid)"); got != "Magic Initiate (Druid)" {
		t.Errorf("titleCase = %q", got)
	}
}
