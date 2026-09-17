import type {Dispatch, SetStateAction} from 'react';
import type {ViewState} from './history-workspace-view-state';

type SelectionArgs = {
    setView: Dispatch<SetStateAction<ViewState>>;
    selectedIds: string[];
    visibleIds: string[];
    visibleSuggestedIds: string[];
    strategy: ViewState['strategy'];
    projectChoices: string[];
};

export function useSelectionActions(args: SelectionArgs) {
    const chooseStrategy = (strategy: ViewState['strategy']) => {
        args.setView((current) => ({
            ...current,
            strategy,
            manualSelectedIds: strategy === 'manual' ? args.selectedIds : current.manualSelectedIds,
            selectedProject: strategy === 'project' && current.selectedProject === '' ? args.projectChoices[0] ?? '' : current.selectedProject,
        }));
    };
    const toggleSelected = (threadID: string) => {
        if (args.strategy !== 'manual') {
            chooseStrategy('manual');
            args.setView((current) => ({...current, manualSelectedIds: args.selectedIds}));
        }
        args.setView((current) => ({
            ...current,
            manualSelectedIds: current.manualSelectedIds.includes(threadID)
                ? current.manualSelectedIds.filter((id) => id !== threadID)
                : [...current.manualSelectedIds, threadID],
        }));
    };
    const toggleAllVisible = () => {
        const selected = new Set(args.selectedIds);
        const allVisibleSelected = args.visibleIds.every((id) => selected.has(id));
        args.setView((current) => {
            const manualSelected = new Set(args.strategy === 'manual' ? current.manualSelectedIds : args.selectedIds);
            args.visibleIds.forEach((id) => allVisibleSelected ? manualSelected.delete(id) : manualSelected.add(id));
            return {...current, strategy: 'manual', manualSelectedIds: [...manualSelected]};
        });
    };
    const selectSuggested = () => {
        args.setView((current) => ({...current, strategy: 'manual', manualSelectedIds: args.visibleSuggestedIds}));
    };
    return {chooseStrategy, selectSuggested, toggleAllVisible, toggleSelected};
}
