import {
    diagnosisFor,
    duplicateKindLabel,
    type DuplicateAnalysis,
    type ThreadDuplicateDiagnosis,
} from './history-workspace-duplicates';
import {formatBytes, formatDateTime, projectLabel} from './history-workspace-helpers';
import type {HistoryThread} from './history-types';

type MetaTone = 'project' | 'time' | 'size' | 'source' | 'provider' | 'thread' | 'diff';
type MetaItem = { key: string; text: string; tone: MetaTone; variant?: string };

export function HistoryThreadTable({
    allItems,
    items,
    duplicateAnalysis,
    selectedIds,
    toggleAllVisible,
    toggleSelected,
}: {
    allItems: HistoryThread[];
    items: HistoryThread[];
    duplicateAnalysis: DuplicateAnalysis;
    selectedIds: string[];
    toggleAllVisible: () => void;
    toggleSelected: (threadID: string) => void;
}) {
    if (items.length === 0) return <div className="空态 小号">没有匹配会话</div>;
    const selected = new Set(selectedIds);
    const allSelected = items.every((item) => selected.has(item.id));
    const partiallySelected = selected.size > 0 && !allSelected;
    const allById = new Map(allItems.map((item) => [item.id, item]));
    return (
        <div className="表格壳 会话表壳">
            <div className="列表表头 列表表头-对比">
                <label className="行勾选头">
                    <input
                        aria-label="全选当前筛选结果"
                        checked={allSelected}
                        onChange={toggleAllVisible}
                        ref={(input) => { if (input) input.indeterminate = partiallySelected; }}
                        type="checkbox"
                    />
                </label>
                <span>当前会话</span>
                <span>对比会话</span>
                <span>建议</span>
            </div>
            <div aria-label="会话列表" className="会话列表" role="region" tabIndex={0}>
                {items.map((item) => (
                    <ThreadRow
                        allById={allById}
                        diagnosis={diagnosisFor(duplicateAnalysis, item.id)}
                        isSelected={selected.has(item.id)}
                        item={item}
                        key={item.id}
                        toggleSelected={toggleSelected}
                    />
                ))}
            </div>
        </div>
    );
}

function ThreadRow({
    allById,
    diagnosis,
    isSelected,
    item,
    toggleSelected,
}: {
    allById: Map<string, HistoryThread>;
    diagnosis: ThreadDuplicateDiagnosis;
    isSelected: boolean;
    item: HistoryThread;
    toggleSelected: (threadID: string) => void;
}) {
    const partner = diagnosis.partnerId ? allById.get(diagnosis.partnerId) ?? null : null;
    return (
        <label className={`会话行 会话行-对比 ${isSelected ? '已选行' : ''}`}>
            <div className="会话勾选">
                <input
                    aria-label={`选择会话 ${item.title || shortID(item.id)}`}
                    checked={isSelected}
                    onChange={() => toggleSelected(item.id)}
                    type="checkbox"
                />
            </div>
            <ThreadCard item={item}/>
            <CompareCard diagnosis={diagnosis} item={item} partner={partner}/>
            <SuggestionCard diagnosis={diagnosis}/>
        </label>
    );
}

function ThreadCard({item}: { item: HistoryThread }) {
    const preview = threadPreview(item);
    return (
        <div className="会话主体">
            <div className="候选主值" title={item.title || '未命名会话'}>{item.title || '未命名会话'}</div>
            <div className="候选副值" title={preview}>{preview}</div>
            <div className="会话元信息">
                {rowMeta(item).map((meta) => <MetaTag key={`${item.id}:${meta.key}`} meta={meta}/>)}
            </div>
        </div>
    );
}

function CompareCard({
    diagnosis,
    item,
    partner,
}: {
    diagnosis: ThreadDuplicateDiagnosis;
    item: HistoryThread;
    partner: HistoryThread | null;
}) {
    if (!partner || diagnosis.kind === 'none') {
        return <div className="对比空态">未发现需要对比的会话</div>;
    }
    return (
        <div className="会话主体 对比主体">
            <div className="候选主值" title={partner.title || '未命名会话'}>{partner.title || '未命名会话'}</div>
            <div className="候选副值" title={threadPreview(partner)}>{threadPreview(partner)}</div>
            <div className="会话元信息">
                {rowMeta(partner).map((meta) => <MetaTag key={`${partner.id}:${meta.key}`} meta={meta}/>)}
            </div>
            <div className="会话元信息 对比差异">
                {diffMeta(item, partner, diagnosis).map((meta) => <MetaTag key={`${partner.id}:${meta.key}`} meta={meta}/>)}
            </div>
        </div>
    );
}

function SuggestionCard({diagnosis}: { diagnosis: ThreadDuplicateDiagnosis }) {
    const action = actionBadge(diagnosis);
    return (
        <div className="会话状态 建议列">
            <span className={`状态签 ${action.tone}`}>{action.label}</span>
            {diagnosis.kind !== 'none' ? <span className="状态签 neutral">{duplicateKindLabel(diagnosis.kind)}</span> : null}
            <div className="判定说明">{diagnosis.reason || '未命中重复标准，建议保留。'}</div>
            {diagnosis.groupSize > 1 ? <div className="判定补充">{diagnosis.groupSize} 条同首消息会话</div> : null}
        </div>
    );
}

function MetaTag({meta}: { meta: MetaItem }) {
    return (
        <span
            className={`会话元信息项 ${meta.tone}${meta.variant ? ` ${meta.tone}-${meta.variant}` : ''}`}
            title={meta.text}
        >
            {meta.text}
        </span>
    );
}

function rowMeta(item: HistoryThread) {
    const items: MetaItem[] = [
        {key: 'project', text: projectLabel(item), tone: 'project'},
        {key: 'updatedAt', text: formatDateTime(item.updatedAt), tone: 'time'},
        {key: 'size', text: formatBytes(item.sizeBytes), tone: 'size'},
    ];
	if (item.isClone) items.push({key: 'metadata-clone', text: `元数据克隆 · ${shortID(item.clonedFrom)}`, tone: 'diff'});
	if (!item.registered) items.push({key: 'file-clone', text: '文件型', tone: 'source'});
    const source = sourceLabel(item);
    if (source !== '') items.push({key: 'source', text: source, tone: 'source'});
    const provider = providerLabel(item.modelProvider);
    if (provider !== '') items.push({key: 'provider', text: provider, tone: 'provider', variant: providerVariant(item.modelProvider)});
    const threadSource = threadSourceLabel(item.threadSource);
    if (threadSource !== '') items.push({key: 'threadSource', text: threadSource, tone: 'thread'});
    return items;
}

function diffMeta(item: HistoryThread, partner: HistoryThread, diagnosis: ThreadDuplicateDiagnosis) {
    const items: MetaItem[] = [{key: 'first-message', text: '首条消息一致', tone: 'diff'}];
    if (diagnosis.kind === 'similar') items.push({key: 'size-diff', text: `保留项更大 ${formatBytes(Math.abs(partner.sizeBytes - item.sizeBytes))}`, tone: 'diff'});
    if (providerLabel(item.modelProvider) !== providerLabel(partner.modelProvider)) {
        items.push({key: 'provider-diff', text: `${providerLabel(item.modelProvider)} -> ${providerLabel(partner.modelProvider)}`, tone: 'diff'});
    }
    if (projectLabel(item) !== projectLabel(partner)) items.push({key: 'project-diff', text: '项目目录不同', tone: 'diff'});
    if (formatDateTime(item.updatedAt) !== formatDateTime(partner.updatedAt)) {
        items.push({key: 'time-diff', text: `保留较新：${formatDateTime(partner.updatedAt)}`, tone: 'diff'});
    }
    return items;
}

function actionBadge(diagnosis: ThreadDuplicateDiagnosis) {
    if (diagnosis.disposition === 'delete') return {label: '建议删除', tone: 'warn' as const};
    if (diagnosis.disposition === 'keep') return {label: '建议保留', tone: 'accent' as const};
    return {label: '建议保留', tone: 'neutral' as const};
}

function threadPreview(item: HistoryThread) {
    return item.preview.trim() || item.firstUserMessage.trim() || item.sourceTitle.trim() || '无预览';
}

function sourceLabel(item: HistoryThread) {
    if (item.threadSource === 'subagent') return '子代理';
    if (item.source.trim().startsWith('{')) return '子代理';
    if (item.source === 'vscode') return 'VSCode';
    if (item.source === 'cli') return 'CLI';
    return item.source.trim();
}

function providerLabel(value: string) {
    if (value === 'hi_code') return 'Hi Code';
    if (value === 'openai') return 'OpenAI';
    if (value === 'custom') return '自定义';
    return value.trim();
}

function providerVariant(value: string) {
    if (value === 'hi_code') return 'hi-code';
    if (value === 'openai') return 'openai';
    if (value === 'custom') return 'custom';
    return 'generic';
}

function threadSourceLabel(value: string) {
    if (value === 'subagent') return '派生会话';
    return '';
}

function shortID(value: string) {
    return value.slice(0, 8);
}
