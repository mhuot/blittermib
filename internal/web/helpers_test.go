package web

import (
	"testing"

	"github.com/no42-org/blittermib/internal/model"
)

func TestOIDUnderPrefix(t *testing.T) {
	cases := []struct {
		name   string
		oid    string
		prefix string
		want   bool
	}{
		{"empty prefix matches anything", "1.3.6.1", "", true},
		{"empty oid matches empty prefix", "", "", true},
		{"empty oid does not match non-empty prefix", "", "1.3", false},
		{"exact match counts as under", "1.3", "1.3", true},
		{"strict prefix with dot boundary", "1.3.6", "1.3", true},
		{"deep prefix match", "1.3.6.1.2.1.2.2.1.10", "1.3.6.1.2.1.2.2.1", true},
		// The classic substring-of-different-OID trap. `1.3` must
		// not match `1.30.6` even though it's a byte-prefix.
		{"numeric substring trap rejected", "1.30.6", "1.3", false},
		{"numeric substring trap rejected (single segment)", "10.20", "1", false},
		// Prefix longer than oid is never a match.
		{"prefix longer than oid", "1", "1.3", false},
		// First-segment differences.
		{"different first segment", "2.3.6", "1.3.6", false},
		// Prefix with trailing dot is malformed but should not match.
		{"trailing-dot prefix does not match", "1.3.6", "1.3.", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := OIDUnderPrefix(c.oid, c.prefix)
			if got != c.want {
				t.Errorf("OIDUnderPrefix(%q, %q) = %v, want %v", c.oid, c.prefix, got, c.want)
			}
		})
	}
}

func TestIsEmptyCounts(t *testing.T) {
	if !isEmptyCounts(nil) {
		t.Error("nil FamilyCounts should be empty")
	}
	if !isEmptyCounts(&model.FamilyCounts{}) {
		t.Error("zero-value FamilyCounts should be empty")
	}
	if isEmptyCounts(&model.FamilyCounts{Counters: 1}) {
		t.Error("FamilyCounts with one Counter should not be empty")
	}
	if isEmptyCounts(&model.FamilyCounts{Structs: 1}) {
		t.Error("FamilyCounts with one Struct should not be empty")
	}
}

func TestTrapTypeLetterCommonTCs(t *testing.T) {
	cases := []struct {
		syntax string
		want   string
	}{
		// Base types
		{"INTEGER", "i"},
		{"Integer32", "i"},
		{"Unsigned32", "u"},
		{"Gauge32", "u"},
		{"Counter32", "c"},
		{"Counter64", "C"},
		{"TimeTicks", "t"},
		{"OCTET STRING", "s"},
		{"OBJECT IDENTIFIER", "o"},
		{"BITS", "b"},

		// Common Textual Conventions
		{"IpAddress", "a"},
		{"MacAddress", "x"},
		{"PhysAddress", "x"},
		{"DisplayString", "s"},
		{"SnmpAdminString", "s"},
		{"TimeStamp", "t"},
		{"DateAndTime", "s"},
		// TruthValue and RowStatus are INTEGER subtypes per RFCs
		// 1903 / 2579; the spec mandates "underlying base type's
		// letter", so they map to `i`. Modal UX surfaces an
		// inline hint reminding the user to type the integer.
		{"TruthValue", "i"},
		{"RowStatus", "i"},

		// Inline enum bodies stripped — INTEGER {up(1), down(2)}
		// is an INTEGER for snmptrap purposes.
		{"INTEGER {up(1), down(2)}", "i"},
		{"INTEGER { up(1), down(2), testing(3) }", "i"},

		// Size / range constraints stripped.
		{"Integer32 (1..2147483647)", "i"},
		{"OCTET STRING (SIZE(0..255))", "s"},

		// Whitespace tolerant
		{"  INTEGER  ", "i"},
		{"\tCounter32\n", "c"},

		// Defensive default for unknown vendor TCs
		{"SomeVendorSpecificType", "s"},
		{"", "s"},

		// Compile-layer expansions that show up as varbind syntax
		// in the wild. `Enumeration` is what the smidump-derived
		// model emits for INTEGER-subtype TCs with a named-number
		// list (IfAdminStatus, IfOperStatus, etc.).
		{"Enumeration", "i"},

		// Common integer-subtype TCs — exact TC names as they
		// appear in the syntax field after the compile layer.
		{"InterfaceIndex", "i"},
		{"InterfaceIndexOrZero", "i"},
		{"InetPortNumber", "i"},
		{"InetVersion", "i"},
		{"IANAifType", "i"},
	}
	for _, c := range cases {
		t.Run(c.syntax, func(t *testing.T) {
			if got := TrapTypeLetter(c.syntax); got != c.want {
				t.Errorf("TrapTypeLetter(%q) = %q, want %q", c.syntax, got, c.want)
			}
		})
	}
}

// TestWorkspaceRowURL pins the leaf-vs-container click semantics —
// in particular the parent-scope fallback that runs when a leaf is
// clicked from the unscoped module view. Clicking `linkDown`
// (a notification) from `/m/SNMPv2-MIB` should land on
// `/m/SNMPv2-MIB/{snmpTraps-OID}?sel={linkDown-OID}` so the list
// pane shows the leaf's siblings instead of every symbol in the
// module.
func TestWorkspaceRowURL(t *testing.T) {
	const (
		moduleName   = "SNMPv2-MIB"
		snmpTraps    = "1.3.6.1.6.3.1.1.5"
		linkDownOID  = "1.3.6.1.6.3.1.1.5.3"
		ifEntryOID   = "1.3.6.1.2.1.2.2.1"
		ifIndexOID   = "1.3.6.1.2.1.2.2.1.1"
		topLevelLeaf = "1.3.6.1.4.1.99999"
	)
	moduleView := &WorkspaceView{Module: &model.Module{Name: moduleName}}
	ifEntryScopedView := &WorkspaceView{
		Module:   &model.Module{Name: moduleName},
		ScopeOID: ifEntryOID,
	}
	interfacesScopedView := &WorkspaceView{
		Module:   &model.Module{Name: moduleName},
		ScopeOID: "1.3.6.1.2.1.2",
	}

	cases := []struct {
		name string
		view *WorkspaceView
		sym  *model.Symbol
		want string
	}{
		{
			"nil symbol falls back to module page",
			moduleView,
			nil,
			"/m/" + moduleName,
		},
		{
			"leaf with no scope and a parent OID scopes to the parent",
			moduleView,
			&model.Symbol{
				ModuleName: moduleName, Name: "linkDown",
				OID: linkDownOID, ParentOID: snmpTraps,
				Kind: model.KindNotificationType,
			},
			"/m/" + moduleName + "/" + snmpTraps + "?sel=" + linkDownOID,
		},
		{
			"leaf inside current scope preserves that scope",
			ifEntryScopedView,
			&model.Symbol{
				ModuleName: moduleName, Name: "ifIndex",
				OID: ifIndexOID, ParentOID: ifEntryOID,
				Kind: model.KindColumn,
			},
			"/m/" + moduleName + "/" + ifEntryOID + "?sel=" + ifIndexOID,
		},
		{
			"leaf outside current scope switches to the leaf's parent",
			interfacesScopedView,
			&model.Symbol{
				ModuleName: moduleName, Name: "linkDown",
				OID: linkDownOID, ParentOID: snmpTraps,
				Kind: model.KindNotificationType,
			},
			"/m/" + moduleName + "/" + snmpTraps + "?sel=" + linkDownOID,
		},
		{
			"leaf with no scope and no parent OID falls back to module root",
			moduleView,
			&model.Symbol{
				ModuleName: moduleName, Name: "orphanLeaf",
				OID: topLevelLeaf, ParentOID: "",
				Kind: model.KindNotificationType,
			},
			"/m/" + moduleName + "?sel=" + topLevelLeaf,
		},
		{
			"container drills in (scope change)",
			moduleView,
			&model.Symbol{
				ModuleName: moduleName, Name: "ifEntry",
				OID: ifEntryOID, ParentOID: "1.3.6.1.2.1.2.2",
				Kind: model.KindTableEntry,
			},
			"/m/" + moduleName + "/" + ifEntryOID,
		},
		{
			"no-OID symbol with no scope rides in by name to module root",
			moduleView,
			&model.Symbol{
				ModuleName: moduleName, Name: "TruthValue",
				Kind: model.KindTextualConvention,
			},
			"/m/" + moduleName + "?sel=TruthValue",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := string(WorkspaceRowURL(c.view, c.sym))
			if got != c.want {
				t.Errorf("WorkspaceRowURL = %q, want %q", got, c.want)
			}
		})
	}
}
