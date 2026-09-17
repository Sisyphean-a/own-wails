import type {HistoryThread} from './history-types';

export type DuplicateKind = 'none' | 'duplicate' | 'similar';
export type DuplicateDisposition = 'none' | 'keep' | 'delete';
export type DiagnosisFilter = 'all' | 'redundant' | 'duplicate' | 'similar' | 'metadata-clone' | 'delete';

export type ThreadDuplicateDiagnosis = {
    kind: DuplicateKind;
    disposition: DuplicateDisposition;
    partnerId: string | null;
    groupKey: string | null;
    groupSize: number;
    reason: string;
};

export type DuplicateAnalysis = {
    byId: Map<string, ThreadDuplicateDiagnosis>;
    summary: {
        groupCount: number;
        redundantCount: number;
        similarCount: number;
        duplicateCount: number;
    };
};

type GroupDiagnosis = {
    keeper: HistoryThread;
    redundant: Array<{ thread: HistoryThread; kind: Exclude<DuplicateKind, 'none'>; reason: string }>;
};

const emptyDiagnosis: ThreadDuplicateDiagnosis = {
    kind: 'none',
    disposition: 'none',
    partnerId: null,
    groupKey: null,
    groupSize: 0,
    reason: '',
};

export function analyzeThreadDuplicates(threads: HistoryThread[]): DuplicateAnalysis {
    const byId = new Map<string, ThreadDuplicateDiagnosis>();
	const summary = {groupCount: 0, redundantCount: 0, similarCount: 0, duplicateCount: 0};
    for (const thread of threads) byId.set(thread.id, emptyDiagnosis);
    for (const group of groupedByFirstMessage(threads)) {
        const diagnosis = buildGroupDiagnosis(group);
        if (!diagnosis) continue;
        applyGroupDiagnosis(byId, summary, diagnosis);
    }
    return {byId, summary};
}

export function diagnosisFor(analysis: DuplicateAnalysis, threadId: string) {
    return analysis.byId.get(threadId) ?? emptyDiagnosis;
}

export function isSuggestedDeleteDiagnosis(diagnosis: ThreadDuplicateDiagnosis) {
    return diagnosis.disposition === 'delete';
}

export function matchesDiagnosisFilter(filter: DiagnosisFilter, diagnosis: ThreadDuplicateDiagnosis, isMetadataClone = false) {
    if (filter === 'all') return true;
	if (filter === 'metadata-clone') return isMetadataClone;
    if (filter === 'redundant') return diagnosis.kind !== 'none';
    if (filter === 'delete') return diagnosis.disposition === 'delete';
    return diagnosis.kind === filter;
}

export function duplicateKindLabel(kind: DuplicateKind) {
    if (kind === 'similar') return '相似项';
    if (kind === 'duplicate') return '重复项';
    return '普通项';
}

function groupedByFirstMessage(threads: HistoryThread[]) {
    const groups = new Map<string, HistoryThread[]>();
    for (const thread of threads) {
        if (thread.isClone) continue;
        const key = normalizeText(thread.firstUserMessage);
        if (key === '') continue;
        groups.set(key, [...(groups.get(key) ?? []), thread]);
    }
    return [...groups.entries()]
        .filter(([, group]) => group.length > 1)
        .map(([groupKey, group]) => ({groupKey, group}));
}

function buildGroupDiagnosis(entry: { groupKey: string; group: HistoryThread[] }): GroupDiagnosis | null {
    const keeper = pickKeeper(entry.group);
    const redundant = entry.group
        .filter((thread) => thread.id !== keeper.id)
        .map((thread) => classifyThread(thread, keeper))
        .filter((item): item is { thread: HistoryThread; kind: Exclude<DuplicateKind, 'none'>; reason: string } => item.kind !== 'none');
    if (redundant.length === 0) return null;
    return {keeper, redundant};
}

function applyGroupDiagnosis(
    byId: Map<string, ThreadDuplicateDiagnosis>,
    summary: DuplicateAnalysis['summary'],
    groupDiagnosis: GroupDiagnosis,
) {
    summary.groupCount += 1;
    const lead = highestPriority(groupDiagnosis.redundant);
    byId.set(groupDiagnosis.keeper.id, {
        kind: lead.kind,
        disposition: 'keep',
        partnerId: lead.thread.id,
        groupKey: normalizeText(groupDiagnosis.keeper.firstUserMessage),
        groupSize: groupDiagnosis.redundant.length + 1,
        reason: keepReason(lead.kind),
    });
    for (const item of groupDiagnosis.redundant) {
        byId.set(item.thread.id, {
            kind: item.kind,
            disposition: 'delete',
            partnerId: groupDiagnosis.keeper.id,
            groupKey: normalizeText(groupDiagnosis.keeper.firstUserMessage),
            groupSize: groupDiagnosis.redundant.length + 1,
            reason: item.reason,
        });
        summary.redundantCount += 1;
        incrementKind(summary, item.kind);
    }
}

function highestPriority(items: GroupDiagnosis['redundant'][number][]) {
    return [...items].sort((left, right) => priorityFor(right.kind) - priorityFor(left.kind))[0];
}

function incrementKind(summary: DuplicateAnalysis['summary'], kind: Exclude<DuplicateKind, 'none'>) {
    if (kind === 'similar') summary.similarCount += 1;
    if (kind === 'duplicate') summary.duplicateCount += 1;
}

function pickKeeper(group: HistoryThread[]) {
    return [...group].sort(compareKeeperPriority)[0];
}

function compareKeeperPriority(left: HistoryThread, right: HistoryThread) {
    if (left.sizeBytes !== right.sizeBytes) return right.sizeBytes - left.sizeBytes;
    const timeDiff = timestamp(right.updatedAt) - timestamp(left.updatedAt);
    if (timeDiff !== 0) return timeDiff;
    if (left.archived !== right.archived) return Number(left.archived) - Number(right.archived);
    return left.id.localeCompare(right.id);
}

function classifyThread(thread: HistoryThread, keeper: HistoryThread) {
    if (thread.sizeBytes !== keeper.sizeBytes) {
        return {thread, kind: 'similar' as const, reason: '首条消息一致，当前条目体积更小，判定为相似项'};
    }
    if (!sameConversationShape(thread, keeper)) {
        return {thread, kind: 'none' as const, reason: ''};
    }
    if (hasClonedIdentity(thread, keeper)) {
		return {thread, kind: 'similar' as const, reason: '首条消息和摘要一致，但来源对象不同，判定为相似项'};
    }
    return {thread, kind: 'duplicate' as const, reason: '首条消息和摘要一致，判定为重复项，建议删除较旧记录'};
}

function sameConversationShape(left: HistoryThread, right: HistoryThread) {
    const previewMatches = normalizeText(left.preview) !== '' && normalizeText(left.preview) === normalizeText(right.preview);
    const titleMatches = normalizeText(left.title || left.sourceTitle) !== ''
        && normalizeText(left.title || left.sourceTitle) === normalizeText(right.title || right.sourceTitle);
    return previewMatches || titleMatches;
}

function hasClonedIdentity(left: HistoryThread, right: HistoryThread) {
    return normalizeText(left.modelProvider) !== normalizeText(right.modelProvider)
        || normalizeText(left.source) !== normalizeText(right.source)
        || normalizeText(left.threadSource) !== normalizeText(right.threadSource)
        || normalizeText(left.cwd) !== normalizeText(right.cwd);
}

function keepReason(kind: Exclude<DuplicateKind, 'none'>) {
    if (kind === 'similar') return '首条消息一致，保留体积更大的这条';
    return '内容一致，保留较新的这条';
}

function priorityFor(kind: Exclude<DuplicateKind, 'none'>) {
    if (kind === 'similar') return 3;
    return 1;
}

function timestamp(value: string) {
    const parsed = Date.parse(value);
    return Number.isNaN(parsed) ? 0 : parsed;
}

function normalizeText(value: string) {
    return value.trim().replace(/\s+/g, ' ').toLowerCase();
}
