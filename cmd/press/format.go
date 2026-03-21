package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// column defines a named column for table output with a fixed width.
type column struct {
	Name  string
	Width int
}

// formatList prints a slice of rows in the requested format.
// Each row is a map[string]any keyed by column name.
// The idField is the key used for --format=ids output.
func formatList(format, idField string, columns []column, rows []map[string]any) {
	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(rows)

	case "csv":
		w := csv.NewWriter(os.Stdout)
		header := make([]string, len(columns))
		for i, c := range columns {
			header[i] = c.Name
		}
		w.Write(header)
		for _, row := range rows {
			record := make([]string, len(columns))
			for i, c := range columns {
				record[i] = formatValue(row[c.Name])
			}
			w.Write(record)
		}
		w.Flush()

	case "ids":
		for _, row := range rows {
			fmt.Println(formatValue(row[idField]))
		}

	default: // table
		if len(rows) == 0 {
			return
		}
		// Print header
		parts := make([]string, len(columns))
		for i, c := range columns {
			parts[i] = fmt.Sprintf("%-*s", c.Width, c.Name)
		}
		fmt.Println(strings.Join(parts, " "))

		totalWidth := 0
		for _, c := range columns {
			totalWidth += c.Width + 1
		}
		fmt.Println(strings.Repeat("-", totalWidth))

		// Print rows
		for _, row := range rows {
			for i, c := range columns {
				val := formatValue(row[c.Name])
				if len(val) > c.Width {
					val = val[:c.Width-1] + "…"
				}
				parts[i] = fmt.Sprintf("%-*s", c.Width, val)
			}
			fmt.Println(strings.Join(parts, " "))
		}
	}
}

// formatSingle prints a single entity as key-value pairs or JSON.
func formatSingle(format string, fields []kv) {
	switch format {
	case "json":
		m := make(map[string]any)
		for _, f := range fields {
			m[f.Key] = f.Value
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(m)

	default: // table (key-value pairs)
		maxKey := 0
		for _, f := range fields {
			if len(f.Key) > maxKey {
				maxKey = len(f.Key)
			}
		}
		for _, f := range fields {
			fmt.Printf("%-*s  %s\n", maxKey, f.Key, f.Value)
		}
	}
}

type kv struct {
	Key   string
	Value string
}

// resolveIDOrString parses an argument as either a numeric ID or a string.
// Returns (id, "") if numeric, or (0, str) if not.
func resolveIDOrString(arg string) (int64, string) {
	id, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return 0, arg
	}
	return id, ""
}

func formatValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case int64:
		return strconv.FormatInt(val, 10)
	case int:
		return strconv.Itoa(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", val)
	}
}
