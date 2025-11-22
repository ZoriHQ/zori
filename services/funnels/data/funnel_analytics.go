package data

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/internal/storage/postgres/models"
	"zori/services/funnels/types"
)

type FunnelAnalyticsData struct {
	clickDb *clickhouse.ClickhouseDB
}

func NewFunnelAnalyticsData(clickDb *clickhouse.ClickhouseDB) *FunnelAnalyticsData {
	return &FunnelAnalyticsData{clickDb: clickDb}
}

func (d *FunnelAnalyticsData) AnalyzeSequentialFunnel(ctx *ctx.Ctx, funnel *models.Funnel, startTime time.Time) (*types.FunnelAnalysisResponse, error) {
	if len(funnel.Steps) == 0 {
		return nil, fmt.Errorf("funnel has no steps")
	}

	conditions := make([]string, len(funnel.Steps))
	for i, step := range funnel.Steps {
		condition, err := d.buildStepCondition(step)
		if err != nil {
			return nil, fmt.Errorf("failed to build condition for step %d: %w", i+1, err)
		}
		conditions[i] = condition
	}

	mode := ""
	if funnel.IsStrict {
		mode = ", 'strict_order'"
	}

	query := fmt.Sprintf(`
		WITH funnel_data AS (
			SELECT
				visitor_id,
				windowFunnel(%d%s)(
					client_timestamp_utc,
					%s
				) AS funnel_step
			FROM events
			WHERE organization_id = ?
				AND project_id = ?
				AND client_timestamp_utc >= ?
				AND client_timestamp_utc <= now()
			GROUP BY visitor_id
		)
		SELECT
			funnel_step,
			count() AS visitors
		FROM funnel_data
		GROUP BY funnel_step
		ORDER BY funnel_step
	`, funnel.WindowSeconds, mode, strings.Join(conditions, ",\n\t\t\t\t\t"))

	rows, err := d.clickDb.Db().Query(ctx, query, funnel.OrganizationID, funnel.ProjectID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to execute funnel query: %w", err)
	}
	defer clickhouse.EnsureClosed(rows)

	stepCounts := make(map[int]uint64)
	var totalVisitors uint64

	for rows.Next() {
		var step int
		var visitors uint64
		if err := rows.Scan(&step, &visitors); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		stepCounts[step] = visitors
		totalVisitors += visitors
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	response := &types.FunnelAnalysisResponse{
		FunnelID:      funnel.ID,
		FunnelName:    funnel.Name,
		TotalVisitors: totalVisitors,
		Steps:         make([]types.FunnelStepResult, len(funnel.Steps)),
		AnalyzedAt:    time.Now(),
	}

	var cumulativeVisitors uint64
	for i := len(funnel.Steps); i >= 1; i-- {
		cumulativeVisitors += stepCounts[i]
	}

	for i, step := range funnel.Steps {
		stepNum := i + 1
		var visitorsAtStep uint64
		for j := stepNum; j <= len(funnel.Steps); j++ {
			visitorsAtStep += stepCounts[j]
		}

		var conversionRate float64
		var dropoffRate float64

		if i == 0 {
			if totalVisitors > 0 {
				conversionRate = float64(visitorsAtStep) / float64(totalVisitors)
				dropoffRate = 1.0 - conversionRate
			}
		} else {
			var prevVisitors uint64
			for j := i; j <= len(funnel.Steps); j++ {
				prevVisitors += stepCounts[j]
			}
			if prevVisitors > 0 {
				conversionRate = float64(visitorsAtStep) / float64(prevVisitors)
				dropoffRate = 1.0 - conversionRate
			}
		}

		response.Steps[i] = types.FunnelStepResult{
			StepOrder:      stepNum,
			Name:           step.Name,
			Visitors:       visitorsAtStep,
			ConversionRate: conversionRate,
			DropoffRate:    dropoffRate,
		}
	}

	if len(funnel.Steps) > 0 {
		lastStepVisitors := stepCounts[len(funnel.Steps)]
		response.ConvertedVisitors = lastStepVisitors
		if totalVisitors > 0 {
			response.OverallConversion = float64(lastStepVisitors) / float64(totalVisitors)
		}
	}

	return response, nil
}

func (d *FunnelAnalyticsData) AnalyzeDepthFunnel(ctx *ctx.Ctx, funnel *models.Funnel, startTime time.Time) (*types.DepthFunnelAnalysisResponse, error) {
	if funnel.DepthConfig == nil {
		return nil, fmt.Errorf("funnel has no depth config")
	}

	var depthConfig types.DepthConfig
	if err := json.Unmarshal(funnel.DepthConfig, &depthConfig); err != nil {
		return nil, fmt.Errorf("failed to parse depth config: %w", err)
	}

	entryCondition, err := d.buildConditionFromStepCondition(&depthConfig.EntryCondition)
	if err != nil {
		return nil, fmt.Errorf("failed to build entry condition: %w", err)
	}

	countCondition := "1=1"
	if depthConfig.CountEvent != "*" {
		countCondition = fmt.Sprintf("event_name = '%s'", depthConfig.CountEvent)
	}

	groupByField := "session_id"
	if depthConfig.Scope == "visitor" {
		groupByField = "visitor_id"
	}

	query := fmt.Sprintf(`
		WITH session_depth AS (
			SELECT
				%s AS scope_id,
				countIf(%s) AS event_count,
				argMin(page_path, client_timestamp_utc) AS entry_page
			FROM events
			WHERE organization_id = ?
				AND project_id = ?
				AND client_timestamp_utc >= ?
				AND client_timestamp_utc <= now()
			GROUP BY scope_id
			HAVING %s
		)
		SELECT
			least(event_count, %d) AS depth,
			count() AS sessions
		FROM session_depth
		GROUP BY depth
		ORDER BY depth
	`, groupByField, countCondition, entryCondition, depthConfig.MaxDepth)

	rows, err := d.clickDb.Db().Query(ctx, query, funnel.OrganizationID, funnel.ProjectID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to execute depth funnel query: %w", err)
	}
	defer clickhouse.EnsureClosed(rows)

	var distribution []types.DepthAnalysisResult
	var totalSessions uint64

	for rows.Next() {
		var result types.DepthAnalysisResult
		if err := rows.Scan(&result.Depth, &result.Visitors); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		distribution = append(distribution, result)
		totalSessions += result.Visitors
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	if distribution == nil {
		distribution = []types.DepthAnalysisResult{}
	}

	return &types.DepthFunnelAnalysisResponse{
		FunnelID:          funnel.ID,
		FunnelName:        funnel.Name,
		TotalSessions:     totalSessions,
		DepthDistribution: distribution,
		AnalyzedAt:        time.Now(),
	}, nil
}

func (d *FunnelAnalyticsData) buildStepCondition(step *models.FunnelStep) (string, error) {
	var condition types.StepCondition
	if err := json.Unmarshal(step.Condition, &condition); err != nil {
		return "", fmt.Errorf("failed to parse step condition: %w", err)
	}

	return d.buildConditionFromStepCondition(&condition)
}

func (d *FunnelAnalyticsData) buildConditionFromStepCondition(condition *types.StepCondition) (string, error) {
	var parts []string

	switch condition.Type {
	case types.ConditionTypeEvent:
		if condition.EventName == nil || *condition.EventName == "" {
			return "", fmt.Errorf("event condition requires event_name")
		}
		parts = append(parts, fmt.Sprintf("event_name = '%s'", d.escapeString(*condition.EventName)))

	case types.ConditionTypePageview:
		parts = append(parts, "event_name = '$pageview'")
		if condition.PagePath != nil && *condition.PagePath != "" {
			parts = append(parts, fmt.Sprintf("page_path = '%s'", d.escapeString(*condition.PagePath)))
		} else if condition.PagePattern != nil && *condition.PagePattern != "" {
			parts = append(parts, fmt.Sprintf("match(page_path, '%s')", d.escapeString(*condition.PagePattern)))
		}

	case types.ConditionTypeClick:
		parts = append(parts, "event_name = '$click'")
		if condition.ClickSelector != nil && *condition.ClickSelector != "" {
			parts = append(parts, fmt.Sprintf("click_element_selector = '%s'", d.escapeString(*condition.ClickSelector)))
		}
		if condition.ClickText != nil && *condition.ClickText != "" {
			parts = append(parts, fmt.Sprintf("click_element_text = '%s'", d.escapeString(*condition.ClickText)))
		}
		if condition.ClickCategory != nil && *condition.ClickCategory != "" {
			parts = append(parts, fmt.Sprintf("click_element_category = '%s'", d.escapeString(*condition.ClickCategory)))
		}

	default:
		return "", fmt.Errorf("unknown condition type: %s", condition.Type)
	}

	for _, filter := range condition.Filters {
		filterCondition, err := d.buildFilterCondition(&filter)
		if err != nil {
			return "", err
		}
		parts = append(parts, filterCondition)
	}

	return strings.Join(parts, " AND "), nil
}

func (d *FunnelAnalyticsData) buildFilterCondition(filter *types.PropertyFilter) (string, error) {
	property := d.escapeString(filter.Property)
	value := d.escapeString(filter.Value)

	switch filter.Operator {
	case types.FilterOperatorEq:
		return fmt.Sprintf("%s = '%s'", property, value), nil
	case types.FilterOperatorNeq:
		return fmt.Sprintf("%s != '%s'", property, value), nil
	case types.FilterOperatorContains:
		return fmt.Sprintf("position(%s, '%s') > 0", property, value), nil
	case types.FilterOperatorRegex:
		return fmt.Sprintf("match(%s, '%s')", property, value), nil
	default:
		return "", fmt.Errorf("unknown filter operator: %s", filter.Operator)
	}
}

func (d *FunnelAnalyticsData) escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}
