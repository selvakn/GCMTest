package store

import (
	"context"
	"database/sql"
	"time"
)

// SuccessRateBucket summarizes delivery success across all rounds sent
// within one time bucket (FR-014).
type SuccessRateBucket struct {
	BucketStart time.Time
	Targeted    int
	Received    int
	SuccessRate float64
}

// ReportStore answers aggregate reporting queries across rounds/deliveries.
type ReportStore struct {
	db *sql.DB
}

// NewReportStore constructs a ReportStore backed by db.
func NewReportStore(db *sql.DB) *ReportStore {
	return &ReportStore{db: db}
}

// SuccessRateByBucket groups rounds by their sent_at, truncated to bucket
// ("hour" or "day"), optionally restricted to [since, until], and reports
// how many of the targeted devices ever received each bucket's rounds.
func (s *ReportStore) SuccessRateByBucket(ctx context.Context, bucket string, since, until *time.Time) ([]SuccessRateBucket, error) {
	format := "%Y-%m-%dT%H:00:00"
	if bucket == "day" {
		format = "%Y-%m-%d"
	}

	var sinceArg, untilArg any
	if since != nil {
		sinceArg = *since
	}
	if until != nil {
		untilArg = *until
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT bucket_start, SUM(targeted), SUM(received)
		FROM (
			SELECT
				strftime(?, r.sent_at) AS bucket_start,
				r.targeted_device_count AS targeted,
				(SELECT COUNT(*) FROM deliveries d WHERE d.round_id = r.id AND d.status != 'pending') AS received
			FROM rounds r
			WHERE (? IS NULL OR r.sent_at >= ?) AND (? IS NULL OR r.sent_at <= ?)
		) per_round
		GROUP BY bucket_start
		ORDER BY bucket_start ASC`,
		format, sinceArg, sinceArg, untilArg, untilArg,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var buckets []SuccessRateBucket
	for rows.Next() {
		var bucketStartRaw string
		var b SuccessRateBucket
		if err := rows.Scan(&bucketStartRaw, &b.Targeted, &b.Received); err != nil {
			return nil, err
		}
		parsed, err := time.Parse("2006-01-02T15:04:05", bucketStartRaw)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", bucketStartRaw)
			if err != nil {
				return nil, err
			}
		}
		b.BucketStart = parsed
		if b.Targeted > 0 {
			b.SuccessRate = float64(b.Received) / float64(b.Targeted)
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}
