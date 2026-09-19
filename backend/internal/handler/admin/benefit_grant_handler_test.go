package admin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafeCSVCellPreventsSpreadsheetFormulaInjection(t *testing.T) {
	for _, value := range []string{"=SUM(1,1)", "+cmd", "-cmd", "@cmd", "\tcmd", "\rcmd"} {
		require.Equal(t, "'"+value, safeCSVCell(value))
	}
	require.Equal(t, "normal@example.com", safeCSVCell("normal@example.com"))
	require.Empty(t, safeCSVCell(""))
}
