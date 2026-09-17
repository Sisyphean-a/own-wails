import type {HistoryWorkspaceController} from './history-workspace-contract';
import {
    createPlanReset,
    useBoundControllerEffects,
    useControllerStore,
    useDerivedWorkspaceData,
    usePlanActions,
    useViewValueSetter,
    useWorkspaceActions,
} from './history-workspace-controller-internals';
import {useSelectionActions} from './history-workspace-selection';
import {
    buildActionSet,
    buildFilters,
    buildOverview,
    buildPlanState,
    buildStrategyState,
} from './history-workspace-controller-builders';

export type {HistoryWorkspaceController} from './history-workspace-contract';

export function useHistoryWorkspaceController(): HistoryWorkspaceController {
    const store = useControllerStore();
    const derived = useDerivedWorkspaceData(store.listResult, store.scanWorkspace, store.view);
    const setViewValue = useViewValueSetter(store.setView);
    const resetPlanArtifacts = createPlanReset(store);
    const workspaceActions = useWorkspaceActions({store, resetPlanArtifacts});
    const selectionActions = useSelectionActions({
        setView: store.setView,
        selectedIds: derived.selectedIds,
        visibleIds: derived.visibleThreads.map((thread) => thread.id),
        visibleSuggestedIds: derived.visibleSuggestedIds,
        strategy: store.view.strategy,
        projectChoices: derived.projectChoices,
    });
    const planActions = usePlanActions({
        store,
        selectedIds: derived.selectedIds,
        selectionSignature: derived.selectionSignature,
        resetPlanArtifacts,
        refreshWorkspace: workspaceActions.refreshWorkspace,
    });
    useBoundControllerEffects({store, derived, view: store.view, startScan: workspaceActions.startScan, resetPlanArtifacts});
    return {
        loading: store.loading,
        error: store.error,
        workspaceConfig: store.workspaceConfig,
        scanWorkspace: store.scanWorkspace,
        listResult: store.listResult,
        visibleThreads: derived.visibleThreads,
        duplicateAnalysis: derived.duplicateAnalysis,
        projectChoices: derived.projectChoices,
        selectedIds: derived.selectedIds,
        confirmPhrase: 'DELETE',
        overview: buildOverview({
            allThreads: derived.allThreads,
            suggestedCount: derived.visibleSuggestedIds.length,
            selectedSize: derived.selectedSize,
            executionResult: store.executionResult,
        }),
        filters: buildFilters(store.view, setViewValue),
        strategyState: buildStrategyState(store.view, setViewValue, selectionActions.chooseStrategy),
        planState: buildPlanState({store, view: store.view, visibleThreads: derived.visibleThreads, selectedIds: derived.selectedIds, unknownCount: derived.unknownCount, setViewValue}),
        actions: buildActionSet(workspaceActions, selectionActions, planActions),
    };
}
