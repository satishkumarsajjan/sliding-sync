package state

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type txnRow struct {
	DeviceID  string `db:"user_id"`
	EventID   string `db:"event_id"`
	TxnID     string `db:"txn_id"`
	Timestamp int64  `db:"ts"`
}

type TransactionsTable struct {
	db *sqlx.DB
}

func NewTransactionsTable(db *sqlx.DB) *TransactionsTable {
	// make sure tables are made
	db.MustExec(`
	CREATE TABLE IF NOT EXISTS syncv3_txns (
		user_id TEXT NOT NULL, -- actually device_id
		event_id TEXT NOT NULL,
		txn_id TEXT NOT NULL,
		ts BIGINT NOT NULL,
		UNIQUE(user_id, event_id)
	);
	`)
	return &TransactionsTable{db}
}

func (t *TransactionsTable) Insert(deviceID string, eventIDToTxnID map[string]string) error {
	ts := time.Now()
	rows := make([]txnRow, 0, len(eventIDToTxnID))
	for eventID, txnID := range eventIDToTxnID {
		rows = append(rows, txnRow{
			EventID:   eventID,
			TxnID:     txnID,
			DeviceID:  deviceID,
			Timestamp: ts.UnixMilli(),
		})
	}
	result, err := t.db.NamedQuery(`
		INSERT INTO syncv3_txns (user_id, event_id, txn_id, ts)
        VALUES (:user_id, :event_id, :txn_id, :ts)`, rows)
	if err == nil {
		result.Close()
	}
	return err
}

func (t *TransactionsTable) Clean(boundaryTime time.Time) error {
	_, err := t.db.Exec(`DELETE FROM syncv3_txns WHERE ts <= $1`, boundaryTime.UnixMilli())
	return err
}

func (t *TransactionsTable) Select(deviceID string, eventIDs []string) (map[string]string, error) {
	result := make(map[string]string, len(eventIDs))
	var rows []txnRow
	err := t.db.Select(&rows, `SELECT event_id, txn_id FROM syncv3_txns WHERE user_id=$1 and event_id=ANY($2)`, deviceID, pq.StringArray(eventIDs))
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.EventID] = row.TxnID
	}
	return result, nil
}
