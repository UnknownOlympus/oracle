package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/UnknownOlympus/olympus-protos/gen/go/scraper/olympus"
	"github.com/UnknownOlympus/oracle/internal/models"
	"github.com/UnknownOlympus/oracle/internal/report"
	"github.com/jackc/pgx/v5"
)

func (b *Bot) formatExcelRows(ctx context.Context, userID int64, from, to time.Time) ([]report.ExcelRow, error) {
	tasks, err := b.repo.GetCompletedTasksByExecutor(ctx, userID, from, to)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []report.ExcelRow{}, nil
		}
		return nil, fmt.Errorf("failed to get completed tasks by executor: %w", err)
	}

	var rows []report.ExcelRow
	for _, task := range tasks {
		row, rowErr := b.getExcelRowsFromTask(ctx, task)
		if rowErr != nil {
			return nil, fmt.Errorf("failed to generate structure of excel rows from task '%d': %w", task.ID, rowErr)
		}

		rows = append(rows, row...)
	}

	return rows, nil
}

func (b *Bot) getExcelRowsFromTask(ctx context.Context, task models.TaskDetails) ([]report.ExcelRow, error) {
	defRow := report.ExcelRow{
		ID:           task.ID,
		Type:         task.Type,
		CreationDate: task.CreationDate,
		Description:  task.Description,
		Address:      task.Address,
	}

	customers, err := b.GetCustomersByTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to get customers by task '%d': %w", task.ID, err)
	}

	if len(customers) == 0 {
		defRow.Customer = "-"
		defRow.Contract = "-"
		defRow.Tariff = "-"
		return []report.ExcelRow{defRow}, nil
	}

	rows := make([]report.ExcelRow, 0, len(customers))
	for _, customer := range customers {
		defRow.Customer = customer.Fullname
		defRow.Contract = customer.Contract
		defRow.Tariff = customer.Tariff

		rows = append(rows, defRow)
	}

	return rows, nil
}

func (b *Bot) GetCustomersByTask(ctx context.Context, task models.TaskDetails) ([]models.Customer, error) {
	taskID := int64(task.ID)

	clients, err := b.repo.GetCustomersByTaskID(ctx, taskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []models.Customer{}, nil
		}
		return nil, fmt.Errorf("failed to get customers data from database, task ID '%d': %w", taskID, err)
	}

	result := make([]models.Customer, 0, len(clients))
	for _, client := range clients {
		perClient, clientErr := b.AddContractToCustomer(ctx, client, task)
		if clientErr != nil {
			return nil, fmt.Errorf("failed to generate customer fields: %w", clientErr)
		}
		result = append(result, perClient)
	}

	return result, nil
}

func (b *Bot) AddContractToCustomer(
	ctx context.Context,
	customer models.Customer,
	task models.TaskDetails,
) (models.Customer, error) {
	var req *olympus.GetAgreementsRequest
	if customer.ID != 0 {
		req = &olympus.GetAgreementsRequest{
			Identifier: &olympus.GetAgreementsRequest_CustomerId{CustomerId: customer.ID},
		}
	} else {
		req = &olympus.GetAgreementsRequest{Identifier: &olympus.GetAgreementsRequest_CustomerName{CustomerName: customer.Fullname}}
	}

	resp, err := b.hermesClient.GetAgreements(ctx, req)
	if err != nil {
		return models.Customer{}, fmt.Errorf("failed to get response from hermes (GetAgreements): %w", err)
	}

	agreements := resp.GetAgreements()
	switch len(agreements) {
	case 0:
		return models.Customer{}, nil
	case 1:
		return convertPbCustomerToModel(agreements[0]), nil
	default:
		for _, agreement := range agreements {
			if task.Address == agreement.GetAddress() {
				return convertPbCustomerToModel(agreement), nil
			}
		}
	}

	return models.Customer{}, nil
}

func convertPbCustomerToModel(pbc *olympus.Agreement) models.Customer {
	return models.Customer{
		ID:       pbc.GetId(),
		Fullname: pbc.GetName(),
		Contract: pbc.GetContract(),
		Address:  pbc.GetAddress(),
		Tariff:   pbc.GetTariff(),
	}
}
