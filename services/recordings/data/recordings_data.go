package data

import (
	"encoding/json"
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/recordings/types"
)

type RecordingsData struct {
	clickDb *clickhouse.ClickhouseDB
}

func NewRecordingsData(clickDb *clickhouse.ClickhouseDB) *RecordingsData {
	return &RecordingsData{
		clickDb: clickDb,
	}
}

func (d *RecordingsData) GetSessionRecordings(ctx *ctx.Ctx, req *types.GetRecordingsRequest) ([]types.SessionRecordingSummary, uint64, error) {
	whereConditions := []string{"project_id = ?"}
	args := []interface{}{req.ProjectID}

	if req.VisitorID != nil && *req.VisitorID != "" {
		whereConditions = append(whereConditions, "visitor_id = ?")
		args = append(args, *req.VisitorID)
	}

	if req.SessionID != nil && *req.SessionID != "" {
		whereConditions = append(whereConditions, "session_id = ?")
		args = append(args, *req.SessionID)
	}

	whereClause := "WHERE " + whereConditions[0]
	for i := 1; i < len(whereConditions); i++ {
		whereClause += " AND " + whereConditions[i]
	}

	countQuery := fmt.Sprintf(`
		SELECT count(DISTINCT session_id)
		FROM session_recordings
		%s
	`, whereClause)

	var totalCount uint64
	countRow := d.clickDb.Db().QueryRow(ctx, countQuery, args...)
	if err := countRow.Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT
			session_id,
			visitor_id,
			min(client_timestamp_utc) as started_at,
			max(client_timestamp_utc) as ended_at,
			sum(event_count) as total_events,
			count() as chunk_count,
			any(page_url) as first_page_url,
			any(host) as host,
			any(user_agent) as user_agent
		FROM session_recordings
		%s
		GROUP BY session_id, visitor_id
		ORDER BY started_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	queryArgs := make([]interface{}, len(args), len(args)+2)
	copy(queryArgs, args)
	queryArgs = append(queryArgs, req.Limit, req.Offset)

	rows, err := d.clickDb.Db().Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query session recordings: %w", err)
	}
	defer clickhouse.EnsureClosed(rows)

	var recordings []types.SessionRecordingSummary
	for rows.Next() {
		var recording types.SessionRecordingSummary
		if err := rows.Scan(
			&recording.SessionID,
			&recording.VisitorID,
			&recording.StartedAt,
			&recording.EndedAt,
			&recording.TotalEvents,
			&recording.ChunkCount,
			&recording.FirstPageURL,
			&recording.Host,
			&recording.UserAgent,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan row: %w", err)
		}
		recordings = append(recordings, recording)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating rows: %w", err)
	}

	if recordings == nil {
		recordings = []types.SessionRecordingSummary{}
	}

	return recordings, totalCount, nil
}

func (d *RecordingsData) GetSessionRecordingEvents(ctx *ctx.Ctx, projectID string, sessionID string) (*types.SessionRecordingDetail, error) {
	query := `
		SELECT
			session_id,
			visitor_id,
			page_url,
			host,
			events,
			event_count,
			chunk_index,
			client_timestamp_utc,
			user_agent,
			ip
		FROM session_recordings
		WHERE project_id = ? AND session_id = ?
		ORDER BY chunk_index ASC, client_timestamp_utc ASC
	`

	rows, err := d.clickDb.Db().Query(ctx, query, projectID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query session recording: %w", err)
	}
	defer clickhouse.EnsureClosed(rows)

	var detail types.SessionRecordingDetail
	var allEvents []any

	isFirst := true
	for rows.Next() {
		var (
			sid        string
			visitorID  string
			pageURL    string
			host       string
			eventsJSON string
			eventCount uint32
			chunkIndex uint32
			timestamp  string
			userAgent  string
			ip         string
		)

		if err := rows.Scan(
			&sid,
			&visitorID,
			&pageURL,
			&host,
			&eventsJSON,
			&eventCount,
			&chunkIndex,
			&timestamp,
			&userAgent,
			&ip,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if isFirst {
			detail.SessionID = sid
			detail.VisitorID = visitorID
			detail.Host = host
			detail.UserAgent = userAgent
			detail.IP = ip
			isFirst = true
		}

		var events []any
		if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal events: %w", err)
		}

		allEvents = append(allEvents, events...)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	if detail.SessionID == "" {
		return nil, fmt.Errorf("session recording not found")
	}

	detail.Events = allEvents

	return &detail, nil
}
