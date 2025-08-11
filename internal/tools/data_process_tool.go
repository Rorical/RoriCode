package tools

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// DataProcessTool handles CSV data processing and analysis
type DataProcessTool struct {
	confirmator Confirmator
}

func (d *DataProcessTool) Name() string {
	return "data_process"
}

func (d *DataProcessTool) Description() string {
	return "Process and analyze CSV data: filter, sort, transform, and generate statistics"
}

func (d *DataProcessTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"operation": map[string]interface{}{
			"type":        "string",
			"description": "Operation to perform: 'read', 'filter', 'sort', 'transform', 'stats', 'head', 'tail'",
			"enum":        []string{"read", "filter", "sort", "transform", "stats", "head", "tail"},
		},
		"path": map[string]interface{}{
			"type":        "string",
			"description": "CSV file path (relative to current working directory)",
		},
		"has_header": map[string]interface{}{
			"type":        "boolean",
			"description": "Whether the CSV file has a header row (default: true)",
		},
		"delimiter": map[string]interface{}{
			"type":        "string",
			"description": "CSV delimiter character (default: ',')",
		},
		"filter": map[string]interface{}{
			"type":        "object",
			"description": "Filter criteria for filter operation",
		},
		"sort_by": map[string]interface{}{
			"type":        "string",
			"description": "Column name or index to sort by",
		},
		"sort_desc": map[string]interface{}{
			"type":        "boolean",
			"description": "Sort in descending order (default: false)",
		},
		"transform": map[string]interface{}{
			"type":        "object",
			"description": "Transformation rules for transform operation",
		},
		"columns": map[string]interface{}{
			"type":        "array",
			"description": "Column names or indices for stats operation",
		},
		"limit": map[string]interface{}{
			"type":        "number",
			"description": "Number of rows to return (for head/tail operations, default: 10)",
		},
		"output_path": map[string]interface{}{
			"type":        "string",
			"description": "Output file path for saving results (optional)",
		},
	}
}

func (d *DataProcessTool) RequiredParameters() []string {
	return []string{"operation", "path"}
}

func (d *DataProcessTool) SetConfirmator(confirmator Confirmator) {
	d.confirmator = confirmator
}

type CSVData struct {
	Records   [][]string
	Headers   []string
	Data      [][]string
	HasHeader bool
}

func (d *DataProcessTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter must be a string")
	}

	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter must be a string")
	}

	// Validate path safety
	if filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be relative, not absolute: %s", path)
	}
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("path cannot contain parent directory references (..): %s", path)
	}

	// Get absolute path
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %v", err)
	}

	fullPath := filepath.Join(cwd, path)

	// Parse options
	hasHeader := true
	if val, exists := args["has_header"]; exists {
		if h, ok := val.(bool); ok {
			hasHeader = h
		}
	}

	delimiter := ","
	if val, exists := args["delimiter"]; exists {
		if d, ok := val.(string); ok && len(d) == 1 {
			delimiter = d
		}
	}

	// Read CSV data
	csvData, err := d.readCSV(fullPath, hasHeader, delimiter)
	if err != nil {
		return nil, err
	}

	switch operation {
	case "read":
		return d.readOperation(csvData, path)
	case "filter":
		filterCriteria := args["filter"]
		return d.filterOperation(csvData, path, filterCriteria)
	case "sort":
		sortBy := args["sort_by"]
		sortDesc := false
		if val, exists := args["sort_desc"]; exists {
			if s, ok := val.(bool); ok {
				sortDesc = s
			}
		}
		return d.sortOperation(csvData, path, sortBy, sortDesc)
	case "transform":
		transformRules := args["transform"]
		return d.transformOperation(csvData, path, transformRules)
	case "stats":
		columns := args["columns"]
		return d.statsOperation(csvData, path, columns)
	case "head":
		limit := 10
		if val, exists := args["limit"]; exists {
			if l, ok := val.(float64); ok {
				limit = int(l)
			}
		}
		return d.headOperation(csvData, path, limit)
	case "tail":
		limit := 10
		if val, exists := args["limit"]; exists {
			if l, ok := val.(float64); ok {
				limit = int(l)
			}
		}
		return d.tailOperation(csvData, path, limit)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

func (d *DataProcessTool) readCSV(path string, hasHeader bool, delimiter string) (*CSVData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = rune(delimiter[0])

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %v", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	csvData := &CSVData{
		Records:   records,
		HasHeader: hasHeader,
	}

	if hasHeader {
		csvData.Headers = records[0]
		csvData.Data = records[1:]
	} else {
		csvData.Data = records
		// Generate numeric headers
		if len(records) > 0 {
			csvData.Headers = make([]string, len(records[0]))
			for i := range csvData.Headers {
				csvData.Headers[i] = fmt.Sprintf("column_%d", i)
			}
		}
	}

	return csvData, nil
}

func (d *DataProcessTool) readOperation(csvData *CSVData, path string) (interface{}, error) {
	return map[string]interface{}{
		"operation":   "read",
		"path":        path,
		"rows":        len(csvData.Data),
		"columns":     len(csvData.Headers),
		"headers":     csvData.Headers,
		"has_header":  csvData.HasHeader,
		"sample_data": d.getSampleData(csvData, 5),
	}, nil
}

func (d *DataProcessTool) filterOperation(csvData *CSVData, path string, filterCriteria interface{}) (interface{}, error) {
	if filterCriteria == nil {
		return nil, fmt.Errorf("filter parameter is required for filter operation")
	}

	// Simple filter implementation - could be enhanced
	filteredData := make([][]string, 0)

	if filterMap, ok := filterCriteria.(map[string]interface{}); ok {
		for _, row := range csvData.Data {
			if d.matchesFilter(row, csvData.Headers, filterMap) {
				filteredData = append(filteredData, row)
			}
		}
	}

	result := &CSVData{
		Headers:   csvData.Headers,
		Data:      filteredData,
		HasHeader: csvData.HasHeader,
	}

	return map[string]interface{}{
		"operation":     "filter",
		"path":          path,
		"original_rows": len(csvData.Data),
		"filtered_rows": len(filteredData),
		"headers":       csvData.Headers,
		"sample_data":   d.getSampleData(result, 5),
	}, nil
}

func (d *DataProcessTool) sortOperation(csvData *CSVData, path string, sortBy interface{}, descending bool) (interface{}, error) {
	if sortBy == nil {
		return nil, fmt.Errorf("sort_by parameter is required for sort operation")
	}

	sortIndex := -1
	if sortByStr, ok := sortBy.(string); ok {
		// Find column index by name
		for i, header := range csvData.Headers {
			if header == sortByStr {
				sortIndex = i
				break
			}
		}
	} else if sortByNum, ok := sortBy.(float64); ok {
		sortIndex = int(sortByNum)
	}

	if sortIndex < 0 || sortIndex >= len(csvData.Headers) {
		return nil, fmt.Errorf("invalid sort column: %v", sortBy)
	}

	// Copy data for sorting
	sortedData := make([][]string, len(csvData.Data))
	copy(sortedData, csvData.Data)

	// Sort the data
	sort.Slice(sortedData, func(i, j int) bool {
		if sortIndex >= len(sortedData[i]) || sortIndex >= len(sortedData[j]) {
			return false
		}
		
		val1 := sortedData[i][sortIndex]
		val2 := sortedData[j][sortIndex]
		
		// Try numeric comparison first
		if num1, err1 := strconv.ParseFloat(val1, 64); err1 == nil {
			if num2, err2 := strconv.ParseFloat(val2, 64); err2 == nil {
				if descending {
					return num1 > num2
				}
				return num1 < num2
			}
		}
		
		// Fall back to string comparison
		if descending {
			return val1 > val2
		}
		return val1 < val2
	})

	result := &CSVData{
		Headers:   csvData.Headers,
		Data:      sortedData,
		HasHeader: csvData.HasHeader,
	}

	return map[string]interface{}{
		"operation":   "sort",
		"path":        path,
		"rows":        len(sortedData),
		"sort_by":     sortBy,
		"descending":  descending,
		"headers":     csvData.Headers,
		"sample_data": d.getSampleData(result, 5),
	}, nil
}

func (d *DataProcessTool) transformOperation(csvData *CSVData, path string, transformRules interface{}) (interface{}, error) {
	if transformRules == nil {
		return nil, fmt.Errorf("transform parameter is required for transform operation")
	}

	// Simple transform implementation - could be enhanced
	transformedData := make([][]string, len(csvData.Data))
	copy(transformedData, csvData.Data)

	result := &CSVData{
		Headers:   csvData.Headers,
		Data:      transformedData,
		HasHeader: csvData.HasHeader,
	}

	return map[string]interface{}{
		"operation":   "transform",
		"path":        path,
		"rows":        len(transformedData),
		"headers":     csvData.Headers,
		"sample_data": d.getSampleData(result, 5),
	}, nil
}

func (d *DataProcessTool) statsOperation(csvData *CSVData, path string, columns interface{}) (interface{}, error) {
	stats := make(map[string]interface{})

	// Get column indices to analyze
	columnIndices := make([]int, 0)
	if columns != nil {
		if colSlice, ok := columns.([]interface{}); ok {
			for _, col := range colSlice {
				if colStr, ok := col.(string); ok {
					for i, header := range csvData.Headers {
						if header == colStr {
							columnIndices = append(columnIndices, i)
							break
						}
					}
				} else if colNum, ok := col.(float64); ok {
					columnIndices = append(columnIndices, int(colNum))
				}
			}
		}
	} else {
		// Analyze all columns
		for i := range csvData.Headers {
			columnIndices = append(columnIndices, i)
		}
	}

	// Calculate statistics for each column
	for _, colIndex := range columnIndices {
		if colIndex < 0 || colIndex >= len(csvData.Headers) {
			continue
		}

		colName := csvData.Headers[colIndex]
		colStats := d.calculateColumnStats(csvData.Data, colIndex)
		stats[colName] = colStats
	}

	return map[string]interface{}{
		"operation": "stats",
		"path":      path,
		"rows":      len(csvData.Data),
		"columns":   len(csvData.Headers),
		"headers":   csvData.Headers,
		"stats":     stats,
	}, nil
}

func (d *DataProcessTool) headOperation(csvData *CSVData, path string, limit int) (interface{}, error) {
	endIndex := limit
	if endIndex > len(csvData.Data) {
		endIndex = len(csvData.Data)
	}

	headData := csvData.Data[:endIndex]
	result := &CSVData{
		Headers:   csvData.Headers,
		Data:      headData,
		HasHeader: csvData.HasHeader,
	}

	return map[string]interface{}{
		"operation": "head",
		"path":      path,
		"limit":     limit,
		"returned":  len(headData),
		"headers":   csvData.Headers,
		"data":      d.getSampleData(result, limit),
	}, nil
}

func (d *DataProcessTool) tailOperation(csvData *CSVData, path string, limit int) (interface{}, error) {
	startIndex := len(csvData.Data) - limit
	if startIndex < 0 {
		startIndex = 0
	}

	tailData := csvData.Data[startIndex:]
	result := &CSVData{
		Headers:   csvData.Headers,
		Data:      tailData,
		HasHeader: csvData.HasHeader,
	}

	return map[string]interface{}{
		"operation": "tail",
		"path":      path,
		"limit":     limit,
		"returned":  len(tailData),
		"headers":   csvData.Headers,
		"data":      d.getSampleData(result, limit),
	}, nil
}

func (d *DataProcessTool) getSampleData(csvData *CSVData, limit int) []map[string]interface{} {
	sample := make([]map[string]interface{}, 0)
	
	maxRows := limit
	if maxRows > len(csvData.Data) {
		maxRows = len(csvData.Data)
	}

	for i := 0; i < maxRows; i++ {
		row := make(map[string]interface{})
		for j, header := range csvData.Headers {
			if j < len(csvData.Data[i]) {
				row[header] = csvData.Data[i][j]
			}
		}
		sample = append(sample, row)
	}

	return sample
}

func (d *DataProcessTool) matchesFilter(row []string, headers []string, filter map[string]interface{}) bool {
	// Simple filter matching - could be enhanced
	for column, criteria := range filter {
		colIndex := -1
		for i, header := range headers {
			if header == column {
				colIndex = i
				break
			}
		}
		
		if colIndex < 0 || colIndex >= len(row) {
			return false
		}
		
		value := row[colIndex]
		if criteriaStr, ok := criteria.(string); ok {
			if !strings.Contains(strings.ToLower(value), strings.ToLower(criteriaStr)) {
				return false
			}
		}
	}
	
	return true
}

func (d *DataProcessTool) calculateColumnStats(data [][]string, colIndex int) map[string]interface{} {
	stats := make(map[string]interface{})
	
	values := make([]string, 0)
	numbers := make([]float64, 0)
	
	for _, row := range data {
		if colIndex < len(row) {
			value := row[colIndex]
			values = append(values, value)
			
			if num, err := strconv.ParseFloat(value, 64); err == nil {
				numbers = append(numbers, num)
			}
		}
	}
	
	stats["count"] = len(values)
	stats["non_empty"] = d.countNonEmpty(values)
	stats["unique"] = d.countUnique(values)
	
	if len(numbers) > 0 {
		sort.Float64s(numbers)
		stats["numeric_count"] = len(numbers)
		stats["min"] = numbers[0]
		stats["max"] = numbers[len(numbers)-1]
		stats["mean"] = d.calculateMean(numbers)
		stats["median"] = d.calculateMedian(numbers)
	}
	
	return stats
}

func (d *DataProcessTool) countNonEmpty(values []string) int {
	count := 0
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			count++
		}
	}
	return count
}

func (d *DataProcessTool) countUnique(values []string) int {
	unique := make(map[string]bool)
	for _, v := range values {
		unique[v] = true
	}
	return len(unique)
}

func (d *DataProcessTool) calculateMean(numbers []float64) float64 {
	sum := 0.0
	for _, n := range numbers {
		sum += n
	}
	return sum / float64(len(numbers))
}

func (d *DataProcessTool) calculateMedian(numbers []float64) float64 {
	n := len(numbers)
	if n%2 == 0 {
		return (numbers[n/2-1] + numbers[n/2]) / 2
	}
	return numbers[n/2]
}