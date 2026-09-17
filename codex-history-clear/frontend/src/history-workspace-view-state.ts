import type {AgeFilter, ArchivedFilter, CleanupStrategy, SizeFilter} from './history-workspace-helpers';
import type {DiagnosisFilter} from './history-workspace-duplicates';

export type ViewState = {
    titleQuery: string;
    projectQuery: string;
    archivedFilter: ArchivedFilter;
    ageFilter: AgeFilter;
    sizeFilter: SizeFilter;
    diagnosisFilter: DiagnosisFilter;
    strategy: CleanupStrategy;
    selectedProject: string;
    autoBackup: boolean;
    keepRecent: boolean;
    generateReport: boolean;
    skipUnknown: boolean;
    manualSelectedIds: string[];
    confirmText: string;
};

export function initialViewState(): ViewState {
    return {
        titleQuery: '',
        projectQuery: '',
        archivedFilter: 'all',
        ageFilter: 'any',
        sizeFilter: 'any',
        diagnosisFilter: 'all',
        strategy: 'manual',
        selectedProject: '',
        autoBackup: true,
        keepRecent: true,
        generateReport: true,
        skipUnknown: true,
        manualSelectedIds: [],
        confirmText: '',
    };
}
