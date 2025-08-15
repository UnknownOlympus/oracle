package report

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/UnknownOlympus/oracle/internal/models"
	"github.com/UnknownOlympus/oracle/internal/repository"
	"github.com/xuri/excelize/v2"
)

var ErrNoTasks = errors.New("failed to generate report, 0 task were provided")

// Generator holds the state for the Excel report generation process.
type Generator struct {
	file *excelize.File
}

// NewGenerator creates a n ew report generator.
func NewGenerator() *Generator {
	return &Generator{
		file: excelize.NewFile(),
	}
}

// GenerateExcelReport generates an Excel report for completed tasks executed by a specific user
// within a given date range. It retrieves tasks from the repository, organizes them by type,
// and formats them into an Excel file with appropriate headers and styles. If no tasks are found,
// it returns nil. The function returns a bytes.Buffer containing the Excel file or an error if
// any operation fails.
//
// Parameters:
// - ctx: The context for managing cancellation and deadlines.
// - repo: The repository interface for accessing task data.
// - telegramID: The ID of the user whose tasks are to be reported.
// - from: The start date for filtering tasks.
// - to: The end date for filtering tasks.
//
// Returns:
// - A pointer to a bytes.Buffer containing the Excel report, or nil if no tasks are found.
// - An error if any operation fails during the report generation.
func GenerateExcelReport(
	ctx context.Context,
	repo repository.Interface,
	telegramID int64,
	from, to time.Time,
) (*bytes.Buffer, error) {
	var err error

	tasks, err := repo.GetCompletedTasksByExecutor(ctx, telegramID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks from repo: %w", err)
	}

	if len(tasks) == 0 {
		return nil, ErrNoTasks
	}

	tasksByType := make(map[string][]models.TaskDetails)
	for _, task := range tasks {
		tasksByType[task.Type] = append(tasksByType[task.Type], task)
	}

	gen := NewGenerator()
	defer gen.file.Close()

	if err = gen.addSheets(tasksByType); err != nil {
		return nil, fmt.Errorf("failed to add sheets: %w", err)
	}

	// setup first sheet as active
	gen.file.SetActiveSheet(0)

	// delete default sheet
	if sheetIndex, _ := gen.file.GetSheetIndex("Sheet1"); sheetIndex != -1 {
		if err = gen.file.DeleteSheet("Sheet1"); err != nil {
			return nil, fmt.Errorf("failed to delete default sheet 'Sheet1': %w", err)
		}
	}

	buffer, err := gen.file.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write data from saved file: %w", err)
	}

	return buffer, nil
}

// addSheets adds new sheets to the generator's file based on the provided
// tasksByType map. Each key in the map represents a task type, and the
// corresponding value is a slice of TaskDetails. The function creates a
// new sheet for each task type, sets up the sheet, and populates it with
// the task details. It returns an error if any operation fails during
// the process.
func (g *Generator) addSheets(tasksByType map[string][]models.TaskDetails) error {
	var err error
	headerIndex := 2

	for taskType, tasksInType := range tasksByType {
		sheetName := truncateSheetName(taskType)

		if _, err = g.file.NewSheet(sheetName); err != nil {
			return fmt.Errorf("failed to generate new sheet '%s': %w", sheetName, err)
		}

		if err = g.setupSheet(sheetName, len(tasksInType)); err != nil {
			return fmt.Errorf("failed to setup sheet '%s': %w", sheetName, err)
		}

		// Fill data
		for i, task := range tasksInType {
			if err = g.addRow(sheetName, i+headerIndex, task); err != nil { // i+2, becase the first row - header
				return fmt.Errorf("failed to add row '%d': %w", i+headerIndex, err)
			}
		}
	}
	return nil
}

// setupSheet initializes the specified sheet with headers, styles, and column widths.
// It creates a header style, sets the row height for the headers, and populates the headers
// in the first row. It also configures the width for each column and adds a table to the sheet.
//
// Parameters:
// - sheetName: The name of the sheet to set up.
// - taskCount: The number of tasks to determine the range of the table.
//
// Returns:
// - error: An error if any operation fails, otherwise returns nil.
func (g *Generator) setupSheet(sheetName string, taskCount int) error {
	var err error

	// Style creating
	headerStyle, err := g.file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4F81BD"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center", Horizontal: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create new style: %w", err)
	}

	// Headers creating
	rowHeighnt := 20
	headers := []string{"Task ID", "Creation Date", "Description", "Address", "Customer"}
	if err = g.file.SetRowHeight(sheetName, 1, float64(rowHeighnt)); err != nil {
		return fmt.Errorf("failed to set row height for headers: %w", err)
	}
	if err = g.file.SetSheetRow(sheetName, "A1", &headers); err != nil {
		return fmt.Errorf("failed to set sheet row for headers: %w", err)
	}
	if err = g.file.SetCellStyle(sheetName, "A1", "E1", headerStyle); err != nil {
		return fmt.Errorf("failed to set cell style for headers: %w", err)
	}

	// Setup width column
	widths := map[string]float64{"A": 15, "B": 18, "C": 50, "D": 40, "E": 30} //nolint:mnd // const values for row width
	for col, width := range widths {
		if err = g.file.SetColWidth(sheetName, col, col, width); err != nil {
			return fmt.Errorf("failed to set column width: %w", err)
		}
	}

	// Add table
	if err = g.file.AddTable(sheetName, &excelize.Table{
		Range:     fmt.Sprintf("A1:E%d", taskCount+1),
		Name:      "table_" + strings.ReplaceAll(sheetName, " ", ""),
		StyleName: "TableStyleMedium9",
	}); err != nil {
		return fmt.Errorf("failed to add table: %w", err)
	}

	return nil
}

// addRow adds a new row to the specified sheet with the details of the given task.
// It takes the sheet name, the row number where the data should be added,
// and the task details as parameters. If the operation fails, it returns an error.
func (g *Generator) addRow(sheetName string, rowNum int, task models.TaskDetails) error {
	rowData := []interface{}{
		task.ID,
		task.CreationDate.Format("02.01.2006"),
		task.Description,
		task.Address,
		task.CustomerName,
	}
	cell, _ := excelize.CoordinatesToCellName(1, rowNum)

	if err := g.file.SetSheetRow(sheetName, cell, &rowData); err != nil {
		return fmt.Errorf("failed to set sheet row: %w", err)
	}

	return nil
}

// truncateSheetName truncates the given sheet name to a maximum of 31 runes.
// If the name exceeds 31 runes, it returns the first 31 runes of the name.
// Otherwise, it returns the name as is.
func truncateSheetName(name string) string {
	if utf8.RuneCountInString(name) > 31 {
		runes := []rune(name)
		return string(runes[:31])
	}
	return name
}
