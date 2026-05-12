package db

import "fmt"

// GetAllowedRooms returns all allowed room IDs.
func (d *DB) GetAllowedRooms() ([]int, error) {
	rows, err := d.conn.Query(`SELECT room_id FROM allowed_rooms ORDER BY room_id`)
	if err != nil {
		return nil, fmt.Errorf("query allowed rooms: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan allowed room: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SetAllowedRooms replaces the entire allowed rooms list.
func (d *DB) SetAllowedRooms(roomIDs []int) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM allowed_rooms`); err != nil {
		return fmt.Errorf("clear allowed rooms: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO allowed_rooms (room_id) VALUES (?)`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, id := range roomIDs {
		if _, err := stmt.Exec(id); err != nil {
			return fmt.Errorf("insert allowed room %d: %w", id, err)
		}
	}

	return tx.Commit()
}
