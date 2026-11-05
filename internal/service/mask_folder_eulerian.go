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

type EulerianMasksResult struct {
	Folder       *domain.MaskFolder `json:"folder"`
	MasksCreated int                `json:"masks_created"`
	Analysis     EulerianResult     `json:"analysis"`
}

// CreateEulerianMasks создаёт папку и по одной маске на каждый маршрут
// исходного графа. Предложения рёбер остаются в Analysis и не сохраняются.
// Если маршрутов нет, Folder == nil. Все пути сразу в памяти не накапливаются.
//
// При ошибке/отмене полностью сохранённые маски остаются в папке, а в её
// metadata остаётся complete:false. Недописанная маска удаляется отдельно.
// Метод возвращает частичный результат вместе с ошибкой: вызывающий код
// не должен отбрасывать его. Полная транзакция на весь перебор не используется.
func (mfs *MaskFolderService) CreateEulerianMasks(
	ctx context.Context,
	graphID uuid.UUID,
	limit int,
) (EulerianMasksResult, error) {
	result := EulerianMasksResult{
		Analysis: EulerianResult{
			Kind:       isNotEulirian,
			PathEdges:  []domain.Edge{},
			CycleEdges: []domain.Edge{},
		},
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}

	searchCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	paths := make(chan []uuid.UUID, 8)
	type outcome struct {
		analysis EulerianResult
		err      error
	}
	done := make(chan outcome, 1)
	go func() {
		analysis, err := mfs.maskService.graphService.RunEulerian(searchCtx, graphID, paths, limit)
		// paths уже закрывается внутри RunEulerian / findAllEulirianPathes.
		done <- outcome{analysis: analysis, err: err}
	}()

	// UUID в названии различает повторные запуски при уникальном (graph_id, name).
	runID := uuid.New().String()
	var saveErr error
	for path := range paths {
		if saveErr = searchCtx.Err(); saveErr != nil {
			cancel()
			break
		}

		if result.Folder == nil {
			folder, err := mfs.repo.Create(searchCtx, domain.MaskFolder{
				GraphID:  graphID,
				Name:     "Эйлеровы маршруты — " + runID,
				Metadata: datatypes.JSON(`{"algorithm":"eulerian","complete":false}`),
			})
			if err != nil {
				saveErr = fmt.Errorf("create Eulerian folder: %w", err)
				cancel()
				break
			}
			result.Folder = &folder
		}

		name := fmt.Sprintf("Эйлеров маршрут %d — %s", result.MasksCreated+1, runID)
		mask, err := mfs.maskService.createEulerianPathMask(searchCtx, graphID, name, path)
		if err != nil {
			saveErr = err
			cancel()
			break
		}

		// Папка и маска только что созданы здесь для одного graphID:
		// повторно читать их через AddMember для проверки не требуется.
		_, err = mfs.membersRepo.Create(searchCtx, domain.MaskFolderMember{
			FolderID: result.Folder.ID,
			MaskID:   mask.ID,
		})
		if err != nil {
			cancel()
			saveErr = errors.Join(
				fmt.Errorf("add Eulerian mask %s to folder: %w", mask.ID, err),
				mfs.maskService.deleteIncompleteEulerianMask(ctx, mask.ID),
			)
			break
		}
		result.MasksCreated++
	}

	// При ошибке сохранения cancel разблокирует отправку производителя.
	// Дожидаемся завершения поиска, чтобы не оставить фоновую горутину.
	finished := <-done
	result.Analysis = finished.analysis
	operationErr := finished.err
	if saveErr != nil {
		operationErr = saveErr
		if finished.err != nil && !errors.Is(finished.err, context.Canceled) {
			operationErr = errors.Join(operationErr, finished.err)
		}
	}
	if operationErr == nil {
		operationErr = ctx.Err()
	}
	if result.Folder == nil {
		return result, operationErr
	}

	// Если не удалось сохранить даже одну маску, убираем пустую папку.
	if operationErr != nil && result.MasksCreated == 0 {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cleanupCancel()
		if err := mfs.repo.DeleteByID(cleanupCtx, result.Folder.ID); err != nil {
			operationErr = errors.Join(operationErr, fmt.Errorf("delete empty Eulerian folder: %w", err))
		} else {
			result.Folder = nil
		}
		return result, operationErr
	}

	label := "Эйлеровы маршруты"
	switch result.Analysis.Kind {
	case isEulirian:
		label = "Эйлеровы циклы"
	case isHalfEulirian:
		label = "Эйлеровы пути"
	}
	name := label + " — " + runID
	metadata, err := json.Marshal(struct {
		Algorithm   string    `json:"algorithm"`
		Complete    bool      `json:"complete"`
		Kind        int       `json:"kind"`
		StartNodeID uuid.UUID `json:"start_node_id"`
		MaskCount   int       `json:"mask_count"`
	}{"eulerian", operationErr == nil, result.Analysis.Kind, result.Analysis.StartNodeID, result.MasksCreated})
	if err != nil {
		return result, errors.Join(operationErr, err)
	}

	// При отмене всё равно отмечаем сохранённую часть как незавершённую.
	finalizeCtx := ctx
	if operationErr != nil {
		var finalizeCancel context.CancelFunc
		finalizeCtx, finalizeCancel = context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer finalizeCancel()
	}
	if err := mfs.repo.PatchByID(finalizeCtx, result.Folder.ID, map[string]any{
		"name": name, "metadata": datatypes.JSON(metadata),
	}); err != nil {
		return result, errors.Join(operationErr, fmt.Errorf("finalize Eulerian folder: %w", err))
	}
	result.Folder.Name = name
	result.Folder.Metadata = datatypes.JSON(metadata)
	return result, operationErr
}
