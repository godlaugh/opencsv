package session

import (
	"sort"

	"opencsv/models"
)

// Undo/redo for structural operations, implemented as inverse operations
// (design B): each entry knows how to revert a mutation and, when applied,
// returns the inverse op to push onto the opposite stack. Memory-light — no
// full snapshots except transpose, which reshapes everything.

const maxUndoDepth = 100

type undoOp interface {
	// apply mutates sess to revert/redo and returns the inverse op.
	apply(sess *models.FileSession) undoOp
}

type undoState struct {
	undo []undoOp
	redo []undoOp
}

// --- op: restore specific cell values ---
type cellsOp struct{ cells []models.Cell }

func (o cellsOp) apply(sess *models.FileSession) undoOp {
	inv := make([]models.Cell, 0, len(o.cells))
	for _, c := range o.cells {
		if c.Row < 0 || c.Row >= len(sess.Rows) {
			continue
		}
		row := sess.Rows[c.Row]
		old := ""
		if c.Col < len(row) {
			old = row[c.Col]
		}
		inv = append(inv, models.Cell{Row: c.Row, Col: c.Col, Value: old})
		for len(row) <= c.Col {
			row = append(row, "")
		}
		row[c.Col] = c.Value
		sess.Rows[c.Row] = row
	}
	return cellsOp{cells: inv}
}

// --- op: restore previous row order. scatterOp puts current[i] at perm[i];
//
//	its inverse gathers current[perm[i]] into i. ---
type scatterOp struct{ perm []int }

func (o scatterOp) apply(sess *models.FileSession) undoOp {
	out := make([][]string, len(o.perm))
	for i, p := range o.perm {
		if i < len(sess.Rows) && p >= 0 && p < len(o.perm) {
			out[p] = sess.Rows[i]
		}
	}
	for i := range out {
		if out[i] == nil {
			out[i] = []string{}
		}
	}
	sess.Rows = out
	sess.TotalRows = len(out)
	return gatherOp{perm: o.perm}
}

type gatherOp struct{ perm []int }

func (o gatherOp) apply(sess *models.FileSession) undoOp {
	out := make([][]string, len(o.perm))
	for i, p := range o.perm {
		if p >= 0 && p < len(sess.Rows) {
			out[i] = sess.Rows[p]
		} else {
			out[i] = []string{}
		}
	}
	sess.Rows = out
	sess.TotalRows = len(out)
	return scatterOp{perm: o.perm}
}

// --- op: insert rows (with data) at ascending positions ---
type insertRowsAtOp struct {
	positions []int
	data      [][]string
}

func (o insertRowsAtOp) apply(sess *models.FileSession) undoOp {
	rows := sess.Rows
	for k, pos := range o.positions {
		if pos < 0 {
			pos = 0
		}
		if pos > len(rows) {
			pos = len(rows)
		}
		rows = append(rows[:pos], append([][]string{o.data[k]}, rows[pos:]...)...)
	}
	sess.Rows = rows
	sess.TotalRows = len(rows)
	return deleteRowsOp{positions: append([]int(nil), o.positions...)}
}

// --- op: delete rows at the given indices (captures data for the inverse) ---
type deleteRowsOp struct{ positions []int }

func (o deleteRowsOp) apply(sess *models.FileSession) undoOp {
	pos := append([]int(nil), o.positions...)
	sort.Ints(pos)
	data := make([][]string, len(pos))
	toDel := make(map[int]bool, len(pos))
	for k, p := range pos {
		if p >= 0 && p < len(sess.Rows) {
			data[k] = sess.Rows[p]
		}
		toDel[p] = true
	}
	newRows := make([][]string, 0, len(sess.Rows)-len(pos))
	for i, r := range sess.Rows {
		if !toDel[i] {
			newRows = append(newRows, r)
		}
	}
	sess.Rows = newRows
	sess.TotalRows = len(newRows)
	return insertRowsAtOp{positions: pos, data: data}
}

// --- op: insert columns (with per-row data) at ascending indices ---
type colEntry struct {
	index int
	name  string
	data  []string
}

type insertColsDataOp struct{ cols []colEntry }

func (o insertColsDataOp) apply(sess *models.FileSession) undoOp {
	indices := make([]int, len(o.cols))
	for k, e := range o.cols {
		at := e.index
		if at < 0 {
			at = 0
		}
		if at > len(sess.Columns) {
			at = len(sess.Columns)
		}
		col := models.Column{Index: at, Name: e.name}
		sess.Columns = append(sess.Columns[:at], append([]models.Column{col}, sess.Columns[at:]...)...)
		for r := range sess.Rows {
			v := ""
			if r < len(e.data) {
				v = e.data[r]
			}
			if at > len(sess.Rows[r]) {
				at = len(sess.Rows[r])
			}
			sess.Rows[r] = append(sess.Rows[r][:at], append([]string{v}, sess.Rows[r][at:]...)...)
		}
		indices[k] = e.index
	}
	for i := range sess.Columns {
		sess.Columns[i].Index = i
	}
	return deleteColsOp{indices: indices}
}

// --- op: delete columns at indices (captures names+data for the inverse) ---
type deleteColsOp struct{ indices []int }

func (o deleteColsOp) apply(sess *models.FileSession) undoOp {
	idx := append([]int(nil), o.indices...)
	sort.Ints(idx)
	entries := make([]colEntry, len(idx))
	for k, ci := range idx {
		name := ""
		if ci >= 0 && ci < len(sess.Columns) {
			name = sess.Columns[ci].Name
		}
		data := make([]string, len(sess.Rows))
		for r := range sess.Rows {
			if ci < len(sess.Rows[r]) {
				data[r] = sess.Rows[r][ci]
			}
		}
		entries[k] = colEntry{index: ci, name: name, data: data}
	}
	toDel := make(map[int]bool, len(idx))
	for _, ci := range idx {
		toDel[ci] = true
	}
	newCols := make([]models.Column, 0, len(sess.Columns)-len(idx))
	for _, c := range sess.Columns {
		if !toDel[c.Index] {
			newCols = append(newCols, c)
		}
	}
	sess.Columns = newCols
	for i := range sess.Columns {
		sess.Columns[i].Index = i
	}
	for r := range sess.Rows {
		nr := make([]string, 0, len(sess.Rows[r]))
		for ci, v := range sess.Rows[r] {
			if !toDel[ci] {
				nr = append(nr, v)
			}
		}
		sess.Rows[r] = nr
	}
	return insertColsDataOp{cols: entries}
}

// --- op: replace the columns list (column rename) ---
type colsOp struct{ columns []models.Column }

func (o colsOp) apply(sess *models.FileSession) undoOp {
	prev := append([]models.Column(nil), sess.Columns...)
	sess.Columns = append([]models.Column(nil), o.columns...)
	return colsOp{columns: prev}
}

// --- op: full snapshot (used for transpose, which reshapes everything) ---
type snapshotOp struct {
	rows [][]string
	cols []models.Column
}

func (o snapshotOp) apply(sess *models.FileSession) undoOp {
	inv := snapshotOp{
		rows: sess.Rows,
		cols: append([]models.Column(nil), sess.Columns...),
	}
	sess.Rows = o.rows
	sess.Columns = append([]models.Column(nil), o.cols...)
	sess.TotalRows = len(o.rows)
	return inv
}

// --- Store: recording + undo/redo ---

func (s *Store) record(id string, op undoOp) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.undoStates[id]
	if st == nil {
		st = &undoState{}
		s.undoStates[id] = st
	}
	st.undo = append(st.undo, op)
	if len(st.undo) > maxUndoDepth {
		st.undo = st.undo[len(st.undo)-maxUndoDepth:]
	}
	st.redo = nil // a new action invalidates redo
}

// RecordCells records the OLD cell values so undo can restore them. Call with
// the pre-mutation values of the cells about to change.
func (s *Store) RecordCells(id string, oldCells []models.Cell) {
	s.record(id, cellsOp{cells: oldCells})
}

// RecordPreCells captures current values of the given cell positions, so undo
// restores them. Call BEFORE mutating those cells.
func (s *Store) RecordPreCells(id string, cells []models.Cell) {
	sess, err := s.Get(id)
	if err != nil {
		return
	}
	old := make([]models.Cell, 0, len(cells))
	for _, c := range cells {
		v := ""
		if c.Row >= 0 && c.Row < len(sess.Rows) && c.Col < len(sess.Rows[c.Row]) {
			v = sess.Rows[c.Row][c.Col]
		}
		old = append(old, models.Cell{Row: c.Row, Col: c.Col, Value: v})
	}
	s.record(id, cellsOp{cells: old})
}

// RecordSortPerm records the inverse of a sort. idx is the gather permutation
// used by the sort (newRows[i] = prevRows[idx[i]]); scatterOp{idx} restores the
// previous order.
func (s *Store) RecordSortPerm(id string, idx []int) {
	s.record(id, scatterOp{perm: append([]int(nil), idx...)})
}

// RecordInsertedRows records that blank rows were inserted at [at, at+count).
func (s *Store) RecordInsertedRows(id string, at, count int) {
	pos := make([]int, count)
	for i := 0; i < count; i++ {
		pos[i] = at + i
	}
	s.record(id, deleteRowsOp{positions: pos})
}

// RecordPreDeleteRows captures rows about to be deleted so undo can re-insert
// them. Call BEFORE deletion.
func (s *Store) RecordPreDeleteRows(id string, positions []int) {
	sess, err := s.Get(id)
	if err != nil {
		return
	}
	pos := append([]int(nil), positions...)
	sort.Ints(pos)
	data := make([][]string, len(pos))
	for k, p := range pos {
		if p >= 0 && p < len(sess.Rows) {
			data[k] = append([]string(nil), sess.Rows[p]...)
		} else {
			data[k] = []string{}
		}
	}
	s.record(id, insertRowsAtOp{positions: pos, data: data})
}

// RecordInsertedCols records that blank columns were inserted at [at, at+count).
func (s *Store) RecordInsertedCols(id string, at, count int) {
	idx := make([]int, count)
	for i := 0; i < count; i++ {
		idx[i] = at + i
	}
	s.record(id, deleteColsOp{indices: idx})
}

// RecordPreDeleteCols captures columns about to be deleted. Call BEFORE deletion.
func (s *Store) RecordPreDeleteCols(id string, indices []int) {
	sess, err := s.Get(id)
	if err != nil {
		return
	}
	idx := append([]int(nil), indices...)
	sort.Ints(idx)
	entries := make([]colEntry, len(idx))
	for k, ci := range idx {
		name := ""
		if ci >= 0 && ci < len(sess.Columns) {
			name = sess.Columns[ci].Name
		}
		data := make([]string, len(sess.Rows))
		for r := range sess.Rows {
			if ci < len(sess.Rows[r]) {
				data[r] = sess.Rows[r][ci]
			}
		}
		entries[k] = colEntry{index: ci, name: name, data: data}
	}
	s.record(id, insertColsDataOp{cols: entries})
}

// RecordColumns records the previous columns list (for rename undo). Pass a
// copy of the columns BEFORE the rename.
func (s *Store) RecordColumns(id string, prev []models.Column) {
	s.record(id, colsOp{columns: append([]models.Column(nil), prev...)})
}

// RecordPreTranspose snapshots the full table before a transpose.
func (s *Store) RecordPreTranspose(id string) {
	sess, err := s.Get(id)
	if err != nil {
		return
	}
	rows := make([][]string, len(sess.Rows))
	for i, r := range sess.Rows {
		rows[i] = append([]string(nil), r...)
	}
	s.record(id, snapshotOp{rows: rows, cols: append([]models.Column(nil), sess.Columns...)})
}

// Undo reverts the most recent recorded structural op. Returns false if the
// stack is empty.
func (s *Store) Undo(id string) (bool, error) {
	return s.applyFromStack(id, true)
}

// Redo re-applies the most recently undone op.
func (s *Store) Redo(id string) (bool, error) {
	return s.applyFromStack(id, false)
}

func (s *Store) applyFromStack(id string, isUndo bool) (bool, error) {
	sess, err := s.Get(id)
	if err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.undoStates[id]
	if st == nil {
		return false, nil
	}
	var stack *[]undoOp
	var other *[]undoOp
	if isUndo {
		stack, other = &st.undo, &st.redo
	} else {
		stack, other = &st.redo, &st.undo
	}
	if len(*stack) == 0 {
		return false, nil
	}
	op := (*stack)[len(*stack)-1]
	*stack = (*stack)[:len(*stack)-1]
	inv := op.apply(sess)
	*other = append(*other, inv)
	sess.Modified = true
	sess.DataVersion++
	return true, nil
}
