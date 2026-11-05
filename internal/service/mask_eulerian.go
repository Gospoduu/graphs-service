package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/Gospoduu/graphs-service/internal/domain"
)

// createEulerianPathMask сохраняет один маршрут. Порядок задаётся serial,
// поэтому он не зависит от порядка строк, возвращённых базой данных.
func (ms *MaskService) createEulerianPathMask(
	ctx context.Context,
	graphID uuid.UUID,
	name string,
	path []uuid.UUID,
) (mask domain.Mask, err error) {
	if err := ctx.Err(); err != nil {
		return domain.Mask{}, err
	}
	mask, err = ms.repo.Create(ctx, domain.Mask{
		GraphID:  graphID,
		Name:     name,
		Metadata: datatypes.JSON(`{"is_path":true}`),
	})
	if err != nil {
		return domain.Mask{}, fmt.Errorf("create Eulerian mask: %w", err)
	}

	// Если какой-то участник не сохранился, удаляем недописанную маску.
	// Участников удаляет существующий FK с ON DELETE CASCADE.
	defer func() {
		if err != nil {
			err = errors.Join(err, ms.deleteIncompleteEulerianMask(ctx, mask.ID))
		}
	}()

	for serial, edgeID := range path {
		if err := ctx.Err(); err != nil {
			return mask, err
		}
		metadata, err := json.Marshal(struct {
			Serial int `json:"serial"`
		}{Serial: serial})
		if err != nil {
			return mask, err
		}
		_, err = ms.membersRepo.Create(ctx, domain.MaskMember{
			MaskID:   mask.ID,
			EdgeID:   edgeID,
			Metadata: datatypes.JSON(metadata),
		})
		if err != nil {
			return mask, fmt.Errorf("create member %d of Eulerian mask %s: %w", serial, mask.ID, err)
		}
	}
	return mask, nil
}

func (ms *MaskService) deleteIncompleteEulerianMask(ctx context.Context, id uuid.UUID) error {
	// Отмена запроса не должна мешать очистить недописанный результат.
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := ms.repo.DeleteByID(cleanupCtx, id); err != nil {
		return fmt.Errorf("delete incomplete Eulerian mask %s: %w", id, err)
	}
	return nil
}
