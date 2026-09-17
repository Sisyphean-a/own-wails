import type {CleanupWorkspaceConfig, HistoryPlanResult, HistoryThread} from './history-types';
import type {DiagnosisFilter, DuplicateAnalysis} from './history-workspace-duplicates';
import {diagnosisFor, isSuggestedDeleteDiagnosis, matchesDiagnosisFilter} from './history-workspace-duplicates';

export type CleanupStrategy = 'recommended' | 'conservative' | 'project' | 'manual';
export type ArchivedFilter = 'all' | 'archived' | 'active';
export type AgeFilter = 'any' | '14' | '30' | '90' | '180';
export type SizeFilter = 'any' | '1' | '10' | '50';

type ThreadFilters = {
    titleQuery: string;
    projectQuery: string;
    archivedFilter: ArchivedFilter;
    ageFilter: AgeFilter;
    sizeFilter: SizeFilter;
    diagnosisFilter: DiagnosisFilter;
};

type SelectionOptions = {
    strategy: CleanupStrategy;
    keepRecent: boolean;
    skipUnknown: boolean;
    selectedProject: string;
    manualSelectedIds: string[];
};

const dayMs = 24 * 60 * 60 * 1000;
const megabyte = 1024 * 1024;
const sizeThresholds: Record<SizeFilter, number> = {
    any: 0,
    '1': megabyte,
    '10': 10 * megabyte,
    '50': 50 * megabyte,
};
const ageThresholds: Record<Exclude<AgeFilter, 'any'>, number> = {
	'14': 14,
    '30': 30,
    '90': 90,
    '180': 180,
};

export function formatBytes(bytes: number) {
    if (bytes <= 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let value = bytes;
    let unitIndex = 0;
    while (value >= 1024 && unitIndex < units.length - 1) {
        value /= 1024;
        unitIndex += 1;
    }
    const digits = value >= 10 || unitIndex === 0 ? 0 : 1;
    return `${value.toFixed(digits)} ${units[unitIndex]}`;
}

export function formatDateTime(value: string) {
    const date = new Date(value);
    if (Number.isNaN(date.valueOf())) return value || '无记录';
    return new Intl.DateTimeFormat('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
    }).format(date);
}

export function createFileUrl(path: string) {
    return encodeURI(`file:///${path.replace(/\\/g, '/')}`);
}

export function backupDirectoryForRun(config: CleanupWorkspaceConfig | null, runId: string | undefined) {
    if (!config) return '';
    if (!runId) return config.backupRoot;
    return `${config.backupRoot}\\${runId}\\backup`;
}

export function projectLabel(thread: HistoryThread) {
    return thread.cwd || '未识别项目';
}

export function projectOptions(threads: HistoryThread[]) {
    return Array.from(new Set(threads.map(projectLabel))).sort((left, right) => left.localeCompare(right));
}

export function filterThreads(threads: HistoryThread[], filters: ThreadFilters, duplicateAnalysis: DuplicateAnalysis) {
    return threads.filter((thread) => matchesThread(thread, filters, duplicateAnalysis));
}

export function selectedThreadIds(threads: HistoryThread[], options: SelectionOptions) {
    if (options.strategy === 'manual') {
        const visible = new Set(threads.map((thread) => thread.id));
        return options.manualSelectedIds.filter((id) => visible.has(id));
    }
    return threads.filter((thread) => isSuggestedThread(thread, options)).map((thread) => thread.id);
}

export function selectedProjects(threads: HistoryThread[], selectedIds: string[]) {
    const selected = new Set(selectedIds);
    return projectOptions(threads.filter((thread) => selected.has(thread.id)));
}

export function totalSelectedBytes(threads: HistoryThread[], selectedIds: string[]) {
    const selected = new Set(selectedIds);
    return threads.reduce((total, thread) => total + (selected.has(thread.id) ? thread.sizeBytes : 0), 0);
}

export function latestUpdatedAt(threads: HistoryThread[]) {
    if (threads.length === 0) return '无记录';
    const latest = threads.reduce((current, thread) => {
        const nextTime = Date.parse(thread.updatedAt);
        if (Number.isNaN(nextTime)) return current;
        return nextTime > current ? nextTime : current;
    }, 0);
    return latest === 0 ? '无记录' : formatDateTime(new Date(latest).toISOString());
}

export function buildThreadTags(thread: HistoryThread, selectedIds: string[]) {
    const tags: string[] = [];
    const age = threadAgeDays(thread.updatedAt);
    if (age < 30) tags.push('最近会话');
    if (age >= 90) tags.push('旧会话');
    if (thread.archived) tags.push('已归档');
    if (thread.sizeBytes >= 50 * megabyte) tags.push('大文件');
    tags.push(selectedIds.includes(thread.id) ? '建议清理' : '建议保留');
    if (!thread.rolloutPath) tags.push('未知结构');
    return tags;
}

export function suggestedDeleteThreadIds(threads: HistoryThread[], duplicateAnalysis: DuplicateAnalysis) {
    return threads
        .filter((thread) => isSuggestedDeleteDiagnosis(diagnosisFor(duplicateAnalysis, thread.id)))
        .map((thread) => thread.id);
}

export function buildRiskNotes(args: {
    autoBackup: boolean;
    selectedIds: string[];
    threads: HistoryThread[];
    planResult: HistoryPlanResult | null;
    unknownCount: number;
    keepRecent: boolean;
}) {
    const notes: string[] = [];
    const selected = new Set(args.selectedIds);
    const includesActive = args.threads.some((thread) => selected.has(thread.id) && !thread.archived);
    if (!args.autoBackup) notes.push('已关闭删除前自动备份，真正清理后将不能一键恢复。');
    if (includesActive) notes.push('本次包含未归档会话，删除后将直接改写本地索引。');
    if (args.unknownCount > 0) notes.push(`扫描中发现 ${args.unknownCount} 个未知结构文件，建议先导出报告再确认删除。`);
    if (!args.keepRecent) notes.push('已关闭最近 30 天保护，容易误删刚用过的会话。');
    if ((args.planResult?.warnings.length ?? 0) > 0) notes.push(args.planResult?.warnings[0] ?? '');
    if (notes.length === 0) notes.push('本次按默认安全策略执行，删除前会先备份，删除后可按备份恢复。');
    return notes;
}

function matchesThread(thread: HistoryThread, filters: ThreadFilters, duplicateAnalysis: DuplicateAnalysis) {
    const titleQuery = filters.titleQuery.trim().toLowerCase();
    const projectQuery = filters.projectQuery.trim().toLowerCase();
    const matchesTitle = titleQuery === '' || [thread.title, thread.sourceTitle, thread.preview, thread.firstUserMessage]
        .join('\n')
        .toLowerCase()
        .includes(titleQuery);
    const matchesProject = projectQuery === '' || projectLabel(thread).toLowerCase().includes(projectQuery);
    const diagnosis = diagnosisFor(duplicateAnalysis, thread.id);
    return matchesTitle
        && matchesProject
        && matchesArchived(thread, filters.archivedFilter)
        && matchesAge(thread, filters.ageFilter)
        && thread.sizeBytes >= sizeThresholds[filters.sizeFilter]
		&& matchesDiagnosisFilter(filters.diagnosisFilter, diagnosis, thread.isClone);
}

function matchesArchived(thread: HistoryThread, archivedFilter: ArchivedFilter) {
    if (archivedFilter === 'archived') return thread.archived;
    if (archivedFilter === 'active') return !thread.archived;
    return true;
}

function matchesAge(thread: HistoryThread, ageFilter: AgeFilter) {
    if (ageFilter === 'any') return true;
    return threadAgeDays(thread.updatedAt) >= ageThresholds[ageFilter];
}

function isSuggestedThread(thread: HistoryThread, options: Omit<SelectionOptions, 'manualSelectedIds'>) {
    if (options.keepRecent && threadAgeDays(thread.updatedAt) < 30) return false;
    if (options.skipUnknown && !thread.rolloutPath) return false;
    if (options.strategy === 'recommended') return thread.archived && threadAgeDays(thread.updatedAt) >= 90;
    if (options.strategy === 'conservative') return threadAgeDays(thread.updatedAt) >= 180;
    if (options.strategy === 'project') return options.selectedProject !== '' && projectLabel(thread) === options.selectedProject;
    return false;
}

function threadAgeDays(value: string) {
    const date = new Date(value);
    if (Number.isNaN(date.valueOf())) return 0;
    return Math.max(0, Math.floor((Date.now() - date.valueOf()) / dayMs));
}
