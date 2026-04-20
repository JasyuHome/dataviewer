package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"dataviewer/backend/models"
	"dataviewer/backend/services"

	"github.com/gin-gonic/gin"
)

type ChartHandler struct {
	dbService *services.DatabaseService
}

func NewChartHandler(dbSvc *services.DatabaseService) *ChartHandler {
	return &ChartHandler{dbService: dbSvc}
}

// GenerateChart generates chart data based on parameters
func (h *ChartHandler) GenerateChart(c *gin.Context) {
	var params models.ChartParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Validate table name
	if params.TableName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Table name is required"})
		return
	}

	if !h.dbService.TableExists(params.TableName) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
		return
	}

	// Validate fields
	if params.XField == "" || params.YField == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X and Y fields are required"})
		return
	}

	// Set default limit
	if params.Limit <= 0 {
		params.Limit = 100
	}
	if params.Limit > 1000 {
		params.Limit = 1000
	}

	// Set default chart type
	if params.ChartType == "" {
		params.ChartType = "bar"
	}

	// Build query
	querySQL := fmt.Sprintf(`SELECT "%s", "%s" FROM %s`, params.XField, params.YField, params.TableName)

	if params.Series != "" {
		querySQL = fmt.Sprintf(`SELECT "%s", "%s", "%s" FROM %s`, params.XField, params.YField, params.Series, params.TableName)
	}

	querySQL += fmt.Sprintf(" LIMIT %d", params.Limit)

	rows, err := h.dbService.GetDB().Query(querySQL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed: " + err.Error()})
		return
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	data := make([]map[string]interface{}, 0)

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert numeric values to float64 for charts
			if numVal, ok := toFloat64(val); ok {
				rowMap[col] = numVal
			} else {
				rowMap[col] = fmt.Sprintf("%v", val)
			}
		}
		data = append(data, rowMap)
	}

	// Build chart config based on type
	chartConfig := h.buildChartConfig(params.ChartType, data, params.XField, params.YField)

	c.JSON(http.StatusOK, gin.H{
		"chart_type":  params.ChartType,
		"data":        data,
		"chart_config": chartConfig,
		"x_field":     params.XField,
		"y_field":     params.YField,
	})
}

// buildChartConfig builds ECharts configuration
func (h *ChartHandler) buildChartConfig(chartType string, data []map[string]interface{}, xField, yField string) map[string]interface{} {
	// Extract categories and values
	categories := make([]string, 0)
	values := make([]float64, 0)

	for _, row := range data {
		if xVal, ok := row[xField].(string); ok {
			categories = append(categories, xVal)
		} else if xVal, ok := row[xField].(float64); ok {
			categories = append(categories, fmt.Sprintf("%v", xVal))
		}

		if yVal, ok := row[yField].(float64); ok {
			values = append(values, yVal)
		}
	}

	config := map[string]interface{}{
		"title": map[string]interface{}{
			"text":  "Chart Visualization",
			"left":  "center",
		},
		"tooltip": map[string]interface{}{
			"trigger": "axis",
		},
		"grid": map[string]interface{}{
			"left":   "3%",
			"right":  "4%",
			"bottom": "3%",
			"top":    "60",
			"containLabel": true,
		},
	}

	switch chartType {
	case "line":
		config["xAxis"] = map[string]interface{}{
			"type": "category",
			"data": categories,
		}
		config["yAxis"] = map[string]interface{}{
			"type": "value",
		}
		config["series"] = []map[string]interface{}{
			{
				"name": yField,
				"type": "line",
				"data": values,
				"smooth": true,
			},
		}

	case "bar":
		config["xAxis"] = map[string]interface{}{
			"type": "category",
			"data": categories,
		}
		config["yAxis"] = map[string]interface{}{
			"type": "value",
		}
		config["series"] = []map[string]interface{}{
			{
				"name": yField,
				"type": "bar",
				"data": values,
			},
		}

	case "pie":
		pieData := make([]map[string]interface{}, 0)
		for i, cat := range categories {
			if i < len(values) {
				pieData = append(pieData, map[string]interface{}{
					"name":  cat,
					"value": values[i],
				})
			}
		}
		config["series"] = []map[string]interface{}{
			{
				"name":     yField,
				"type":     "pie",
				"data":     pieData,
				"radius":   "50%",
				"roseType": "area",
			},
		}

	default:
		config["xAxis"] = map[string]interface{}{
			"type": "category",
			"data": categories,
		}
		config["yAxis"] = map[string]interface{}{
			"type": "value",
		}
		config["series"] = []map[string]interface{}{
			{
				"name": yField,
				"type": "bar",
				"data": values,
			},
		}
	}

	return config
}

// toFloat64 tries to convert a value to float64
func toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case nil:
		return 0, false
	case string:
		// Try to parse string as number, removing currency symbols and commas
		cleanVal := strings.ReplaceAll(v, "¥", "")
		cleanVal = strings.ReplaceAll(cleanVal, "$", "")
		cleanVal = strings.ReplaceAll(cleanVal, "€", "")
		cleanVal = strings.ReplaceAll(cleanVal, "£", "")
		cleanVal = strings.ReplaceAll(cleanVal, ",", "")
		cleanVal = strings.TrimSpace(cleanVal)

		if f, err := strconv.ParseFloat(cleanVal, 64); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// GetChartData returns raw data for chart
func (h *ChartHandler) GetChartData(c *gin.Context) {
	tableName := c.Param("tableName")
	chartType := c.Query("type")
	xField := c.Query("x_field")
	yField := c.Query("y_field")
	limit := 100

	if tableName == "" || xField == "" || yField == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters"})
		return
	}

	if !h.dbService.TableExists(tableName) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
		return
	}

	// Build query
	querySQL := fmt.Sprintf(`SELECT "%s", "%s" FROM %s LIMIT %d`, xField, yField, tableName, limit)

	rows, err := h.dbService.GetDB().Query(querySQL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	data := make([]map[string]interface{}, 0)

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if numVal, ok := toFloat64(val); ok {
				// Round to 2 decimal places
				rowMap[col] = math.Round(numVal*100) / 100
			} else {
				rowMap[col] = fmt.Sprintf("%v", val)
			}
		}
		data = append(data, rowMap)
	}

	c.JSON(http.StatusOK, gin.H{
		"chart_type":  chartType,
		"data":        data,
		"x_field":     xField,
		"y_field":     yField,
	})
}
