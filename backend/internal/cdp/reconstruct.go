package cdp

// TableState maps a row key to its most recent serialized row image.
type TableState map[string][]byte

// SnapshotState maps a table name to its reconstructed TableState.
type SnapshotState map[string]TableState

// Reconstruct folds an ordered replay set into the recovered state per table.
//
// Records must be supplied in ascending LSN order (as returned by
// RecoverToLSN and RecoverToTime). Insert and Update store the row's After
// image under its key using last-writer-wins semantics; Delete removes the
// key. Tables that become empty are omitted from the result. Reconstruct does
// not mutate the input slice or the byte slices it references.
func Reconstruct(records []ChangeRecord) SnapshotState {
	state := make(SnapshotState)
	for i := range records {
		rec := records[i]
		switch rec.Op {
		case OpInsert, OpUpdate:
			tbl, ok := state[rec.Table]
			if !ok {
				tbl = make(TableState)
				state[rec.Table] = tbl
			}
			row := make([]byte, len(rec.After))
			copy(row, rec.After)
			tbl[rec.Key] = row
		case OpDelete:
			if tbl, ok := state[rec.Table]; ok {
				delete(tbl, rec.Key)
				if len(tbl) == 0 {
					delete(state, rec.Table)
				}
			}
		}
	}
	return state
}
