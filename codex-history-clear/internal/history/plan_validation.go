package history

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
)

type planStoreIntent struct {
	store  string
	path   string
	action string
	exists bool
}

func validateApprovedPlan(paths codexPaths, approvedPath string, approved planDocument) error {
	if !approved.Approved {
		return fmt.Errorf("删除计划尚未批准，请先生成 approved-plan.json")
	}
	if !sameFilesystemPath(approved.CodexHome, paths.codexHome) {
		return fmt.Errorf("删除计划 CODEX_HOME 与当前目录不匹配")
	}
	if err := validateApprovalSource(approvedPath, approved); err != nil {
		return err
	}
	return validateCurrentPlanIntent(paths, approved)
}

func validateApprovalSource(approvedPath string, approved planDocument) error {
	sourcePath := filepath.Join(filepath.Dir(approvedPath), "delete-plan.json")
	source, err := loadPlanDocument(sourcePath)
	if err != nil {
		return fmt.Errorf("读取原始删除计划失败: %w", err)
	}
	if source.Approved {
		return fmt.Errorf("原始删除计划状态无效")
	}
	approved.Approved = false
	if !reflect.DeepEqual(source, approved) {
		return fmt.Errorf("批准计划与原始删除计划不一致")
	}
	return nil
}

func validateCurrentPlanIntent(paths codexPaths, approved planDocument) error {
	ids := make([]string, 0, len(approved.Targets))
	for _, target := range approved.Targets {
		ids = append(ids, target.Thread.ID)
	}
	currentThreads, err := resolveTargets(paths, ids)
	if err != nil {
		return fmt.Errorf("重建当前删除目标失败: %w", err)
	}
	if err := validateTargetTranscripts(paths, currentThreads); err != nil {
		return err
	}
	currentTargets, _, err := assemblePlanTargets(paths, currentThreads)
	if err != nil {
		return fmt.Errorf("重建当前删除计划失败: %w", err)
	}
	return comparePlanIntents(approved.Targets, currentTargets)
}

func comparePlanIntents(approved []PlanTarget, current []PlanTarget) error {
	if len(approved) != len(current) {
		return fmt.Errorf("批准计划目标数量与当前计划不一致")
	}
	currentByID := make(map[string]PlanTarget, len(current))
	for _, target := range current {
		currentByID[target.Thread.ID] = target
	}
	seen := map[string]struct{}{}
	for _, target := range approved {
		id := target.Thread.ID
		if _, duplicate := seen[id]; duplicate {
			return fmt.Errorf("批准计划包含重复目标: %s", id)
		}
		seen[id] = struct{}{}
		if err := compareTargetIntent(target, currentByID[id]); err != nil {
			return err
		}
	}
	return nil
}

func compareTargetIntent(approved PlanTarget, current PlanTarget) error {
	if current.Thread.ID == "" {
		return fmt.Errorf("当前计划缺少目标: %s", approved.Thread.ID)
	}
	if approved.Thread.IsClone != current.Thread.IsClone ||
		approved.Thread.ClonedFrom != current.Thread.ClonedFrom ||
		approved.Thread.OriginalProvider != current.Thread.OriginalProvider {
		return fmt.Errorf("目标克隆身份已变化: %s", approved.Thread.ID)
	}
	if !reflect.DeepEqual(storeIntents(approved.Stores), storeIntents(current.Stores)) {
		return fmt.Errorf("目标存储计划已变化: %s", approved.Thread.ID)
	}
	return nil
}

func storeIntents(stores []PlanStore) []planStoreIntent {
	intents := make([]planStoreIntent, 0, len(stores))
	for _, store := range stores {
		intents = append(intents, planStoreIntent{
			store: store.Store, path: filepath.Clean(store.Path), action: store.Action, exists: store.Exists,
		})
	}
	sort.Slice(intents, func(i, j int) bool {
		left := intents[i].store + "\x00" + intents[i].path + "\x00" + intents[i].action
		right := intents[j].store + "\x00" + intents[j].path + "\x00" + intents[j].action
		return left < right
	})
	return intents
}

func sameFilesystemPath(left string, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
