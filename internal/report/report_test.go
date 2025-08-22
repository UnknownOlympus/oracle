package report_test

import (
	"testing"
	"time"

	"github.com/UnknownOlympus/oracle/internal/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestGenerateExcelReport(t *testing.T) {
	testRows := []report.ExcelRow{
		{ID: 1, Type: "Type 1", Description: "Task 1", CreationDate: time.Now()},
		{ID: 2, Type: "Type 2", Description: "Task 2", CreationDate: time.Now()},
		{ID: 3, Type: "Type 1", Description: "Task 3", CreationDate: time.Now()},
	}

	t.Run("successful report generation", func(t *testing.T) {
		buffer, err := report.GenerateExcelReport(testRows)

		require.NoError(t, err)
		assert.NotNil(t, buffer)

		f, err := excelize.OpenReader(buffer)
		require.NoError(t, err)
		defer f.Close()

		sheetList := f.GetSheetList()
		assert.ElementsMatch(t, []string{"Type 1", "Type 2"}, sheetList)

		headerVal, err := f.GetCellValue("Type 1", "A1")
		require.NoError(t, err)
		assert.Equal(t, "Task ID", headerVal)

		taskIDVal, err := f.GetCellValue("Type 1", "A2")
		require.NoError(t, err)
		assert.Equal(t, "1", taskIDVal)

		taskDescVal, err := f.GetCellValue("Type 1", "C3")
		require.NoError(t, err)
		assert.Equal(t, "Task 3", taskDescVal)
	})

	t.Run("no tasks found", func(t *testing.T) {
		buffer, err := report.GenerateExcelReport([]report.ExcelRow{})

		require.Error(t, err)
		assert.Nil(t, buffer)
		require.ErrorIs(t, err, report.ErrNoTasks)
	})
}
