// Package storetest provides deterministic corpus fixtures shared by
// unit tests and the e2e harness, so the Go and browser suites exercise
// the same seeded data instead of drifting hand-copied literals.
//
// It is a normal (non-_test) package because cmd/e2e-harness must import
// it; it is never referenced by the production binary.
package storetest

import (
	"context"

	"github.com/no42-org/blittermib/internal/model"
	"github.com/no42-org/blittermib/internal/store"
)

// SeedIFMIB loads a fixed slice of IF-MIB: a table, its entry, and two
// columns. It is enough for a multi-hit text search ("if" matches all
// four symbols), an OID-prefix hit (the 1.3.6.* OIDs answer "1"), and a
// navigable page for each seeded symbol, while staying small enough to
// seed instantly with no smidump.
//
// It also adds one group-member reference from ifPacketGroup, which is
// deliberately NOT seeded as a symbol — mirroring real IF-MIB, where an
// OBJECT-GROUP references objects but isn't itself a browsable column.
// This drives the symbol page's "Used by" panel (asserted by
// server_test.go); the rendered back-link is expected to 404, matching
// the committed golden output. Do not "fix" it by seeding the group
// without updating those expectations.
func SeedIFMIB(ctx context.Context, st *store.Store) error {
	mod := &model.Module{
		Name: "IF-MIB", OIDRoot: "1.3.6.1.2.1.31",
		ParseStatus: model.ParseStatusClean, Description: "Interfaces MIB.",
	}
	syms := []model.Symbol{
		{
			ModuleName: "IF-MIB", Name: "ifTable",
			OID: "1.3.6.1.2.1.2.2", ParentOID: "1.3.6.1.2.1.2",
			Kind: model.KindTable, Syntax: "SEQUENCE OF IfEntry",
			Access: model.AccessNotAccessible, Status: model.StatusCurrent,
			Description: "A list of interface entries.",
		},
		{
			ModuleName: "IF-MIB", Name: "ifEntry",
			OID: "1.3.6.1.2.1.2.2.1", ParentOID: "1.3.6.1.2.1.2.2",
			Kind: model.KindTableEntry, Syntax: "IfEntry",
			Access: model.AccessNotAccessible, Status: model.StatusCurrent,
			IndexColumns: []string{"ifIndex"},
		},
		{
			ModuleName: "IF-MIB", Name: "ifIndex",
			OID: "1.3.6.1.2.1.2.2.1.1", ParentOID: "1.3.6.1.2.1.2.2.1",
			Kind: model.KindColumn, Syntax: "InterfaceIndex",
			Access: model.AccessReadOnly, Status: model.StatusCurrent,
		},
		{
			ModuleName: "IF-MIB", Name: "ifInOctets",
			OID: "1.3.6.1.2.1.2.2.1.10", ParentOID: "1.3.6.1.2.1.2.2.1",
			Kind: model.KindColumn, Syntax: "Counter32",
			Access: model.AccessReadOnly, Status: model.StatusCurrent,
			Units: "octets", Description: "The total number of octets received on the interface.",
		},
	}
	refs := []model.Reference{
		{
			SourceModule: "IF-MIB", SourceName: "ifPacketGroup",
			TargetModule: "IF-MIB", TargetName: "ifInOctets",
			Kind: model.RefGroupMember,
		},
	}
	return st.ReplaceModule(ctx, mod, syms, refs, nil)
}
