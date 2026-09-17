import {
    Archive,
    FolderOpen,
    Funnel,
    GearSix,
    ListChecks,
    MagnifyingGlass,
    Trash,
} from '@phosphor-icons/react';
import type {ReactNode} from 'react';
import {FilterSelect, ProjectFilter} from './history-workspace-filter-controls';
import type {HistoryWorkspaceController} from './history-workspace-controller';

type ToolbarIcon = 'archive' | 'folder' | 'refresh' | 'select' | 'settings' | 'trash';
type ToolbarTone = 'neutral' | 'primary' | 'danger';
type MetricTone = 'accent' | 'warn' | 'neutral';

type ToolbarPanelProps = HistoryWorkspaceController & {
    onOpenPreview: () => void;
    onOpenSettings: () => void;
	onOpenWarnings: () => void;
};

type ToolbarMetric = {
    label: string;
    value: string;
    tone?: MetricTone;
	onClick?: () => void;
};

export function ToolbarPanel(props: ToolbarPanelProps) {
    const previewText = props.loading === 'plan' ? '生成中' : '删除预览';
    return (
        <header className="顶部操作区">
            <div className="应用栏">
                <div className="品牌区">
                    <span className="品牌图标"><Archive aria-hidden="true" size={20} weight="duotone"/></span>
                    <div className="品牌文字">
                        <h1>Codex 会话清理</h1>
                        <div className="目录行">
                            <FolderOpen aria-hidden="true" size={13}/>
                            <code title={props.workspaceConfig?.codexHome}>{props.workspaceConfig?.codexHome ?? '正在读取工作区'}</code>
                        </div>
                    </div>
                </div>
                <div className="顶部操作组">
                    <ToolbarButton icon="settings" onClick={props.onOpenSettings} text="设置"/>
                    <ToolbarButton
                        disabled={props.overview.suggestedCount === 0}
                        icon="select"
                        onClick={props.actions.selectSuggested}
                        text="采用建议"
                        tone="primary"
                    />
                    <ToolbarButton
                        disabled={props.selectedIds.length === 0 || props.loading === 'plan'}
                        icon="trash"
                        onClick={props.onOpenPreview}
                        text={previewText}
                        tone="danger"
                    />
                </div>
            </div>
            <div className="工作区摘要" aria-label="工作区摘要">
                <div className={`扫描状态 ${scanStatusTone(props)}`}>
                    <MagnifyingGlass aria-hidden="true" size={15}/>
                    <span>{scanStatusText(props)}</span>
                </div>
                {toolbarMetrics(props).map((item) => <ToolbarMetricItem key={item.label} {...item}/>)}
            </div>
            <ToolbarFilters {...props}/>
        </header>
    );
}

function ToolbarFilters(props: HistoryWorkspaceController) {
    return (
        <div className="顶部筛选行">
            <div className="筛选标题"><Funnel aria-hidden="true" size={14}/><span>筛选</span></div>
            <ToolbarField label="归档">
                <FilterSelect
                    label="归档筛选"
                    onChange={(value) => props.filters.setArchivedFilter(value as typeof props.filters.archivedFilter)}
                    options={archiveOptions}
                    value={props.filters.archivedFilter}
                />
            </ToolbarField>
            <ToolbarField label="判定">
                <FilterSelect
                    label="重复判定筛选"
                    onChange={(value) => props.filters.setDiagnosisFilter(value as typeof props.filters.diagnosisFilter)}
                    options={diagnosisOptions}
                    value={props.filters.diagnosisFilter}
                />
            </ToolbarField>
            <ToolbarField className="项目筛选" label="项目">
                <ProjectFilter choices={props.projectChoices} onChange={props.filters.setProjectQuery} value={props.filters.projectQuery}/>
            </ToolbarField>
            <ToolbarField label="时间">
                <FilterSelect
                    label="时间筛选"
                    onChange={(value) => props.filters.setAgeFilter(value as typeof props.filters.ageFilter)}
                    options={ageOptions}
                    value={props.filters.ageFilter}
                />
            </ToolbarField>
            <ToolbarField label="大小">
                <FilterSelect
                    label="大小筛选"
                    onChange={(value) => props.filters.setSizeFilter(value as typeof props.filters.sizeFilter)}
                    options={sizeOptions}
                    value={props.filters.sizeFilter}
                />
            </ToolbarField>
        </div>
    );
}

function ToolbarMetricItem({label, value, tone = 'neutral', onClick}: ToolbarMetric & {onClick?: () => void}) {
	if (onClick) {
		return (
			<button className={`顶部信息项 顶部信息按钮 顶部信息项-${tone}`} onClick={onClick} type="button">
				<span>{label}</span><strong title={value}>{value}</strong>
			</button>
		);
	}
    return (
        <div className={`顶部信息项 顶部信息项-${tone}`}>
            <span>{label}</span>
            <strong title={value}>{value}</strong>
        </div>
    );
}

function ToolbarField({label, children, className = ''}: { label: string; children: ReactNode; className?: string }) {
    return (
        <div className={`顶部筛选项 ${className}`}>
            <span>{label}</span>
            {children}
        </div>
    );
}

export function ToolbarButton({text, icon, onClick, disabled, tone = 'neutral'}: {
    text: string;
    icon: ToolbarIcon;
    onClick: () => void | Promise<void>;
    disabled?: boolean;
    tone?: ToolbarTone;
}) {
    const className = tone === 'primary' ? '主按钮' : tone === 'danger' ? '危险按钮' : '次按钮';
    return (
        <button className={`${className} 图标按钮`} disabled={disabled} onClick={onClick} type="button">
            <ToolbarGlyph icon={icon}/><span>{text}</span>
        </button>
    );
}

export function DataRow({label, value, code = false}: { label: string; value: string; code?: boolean }) {
    return <div className="数据行"><span>{label}</span>{code ? <code title={value}>{value}</code> : <strong title={value}>{value}</strong>}</div>;
}

export function scanStatusText(props: HistoryWorkspaceController) {
    if (props.loading === 'scan') return '扫描中';
    if (props.scanWorkspace.kind === 'error') return '扫描失败';
    if (props.scanWorkspace.kind === 'ready') return '扫描完成';
    if (props.listResult) return '已载入';
    return '等待扫描';
}

function ToolbarGlyph({icon}: { icon: ToolbarIcon }) {
    if (icon === 'archive') return <Archive aria-hidden="true" size={15}/>;
    if (icon === 'folder') return <FolderOpen aria-hidden="true" size={15}/>;
    if (icon === 'refresh') return <MagnifyingGlass aria-hidden="true" size={15}/>;
    if (icon === 'select') return <ListChecks aria-hidden="true" size={15}/>;
    if (icon === 'settings') return <GearSix aria-hidden="true" size={15}/>;
    return <Trash aria-hidden="true" size={15}/>;
}

function toolbarMetrics(props: ToolbarPanelProps): ToolbarMetric[] {
    const unknownCount = props.scanWorkspace.kind === 'ready' ? props.scanWorkspace.scan.summary.unknownCount : 0;
    return [
        {label: '会话', value: String(props.overview.totalSessions)},
        {label: '归档', value: String(props.overview.archivedSessions)},
        {label: '重复组', value: String(props.duplicateAnalysis.summary.groupCount)},
        {label: '待确认', value: String(props.overview.suggestedCount), tone: props.overview.suggestedCount > 0 ? 'warn' : 'neutral'},
		{label: '未识别', value: String(unknownCount), tone: unknownCount > 0 ? 'warn' : 'neutral'},
		{label: '扫描告警', value: String(props.listResult?.summary.warningCount ?? 0), tone: (props.listResult?.summary.warningCount ?? 0) > 0 ? 'warn' : 'neutral', onClick: (props.listResult?.summary.warningCount ?? 0) > 0 ? props.onOpenWarnings : undefined},
        {label: '已选择', value: String(props.selectedIds.length), tone: props.selectedIds.length > 0 ? 'accent' : 'neutral'},
        {label: '预计释放', value: props.overview.releaseText, tone: props.selectedIds.length > 0 ? 'accent' : 'neutral'},
    ];
}

function scanStatusTone(props: HistoryWorkspaceController): MetricTone {
    if (props.loading === 'scan' || props.scanWorkspace.kind === 'ready') return 'accent';
    if (props.scanWorkspace.kind === 'error') return 'warn';
    return 'neutral';
}

const archiveOptions = [
    {value: 'all', label: '全部会话'},
    {value: 'archived', label: '仅已归档'},
    {value: 'active', label: '仅未归档'},
] as const;

const diagnosisOptions = [
    {value: 'all', label: '全部判定'},
    {value: 'redundant', label: '仅多余项'},
    {value: 'similar', label: '仅相似项'},
	{value: 'metadata-clone', label: '仅元数据克隆'},
    {value: 'duplicate', label: '仅重复项'},
    {value: 'delete', label: '仅建议删除'},
] as const;

const ageOptions = [
    {value: 'any', label: '全部时间'},
	{value: '14', label: '14 天前'},
    {value: '30', label: '30 天前'},
    {value: '90', label: '90 天前'},
    {value: '180', label: '180 天前'},
] as const;

const sizeOptions = [
    {value: 'any', label: '全部大小'},
    {value: '1', label: '1 MB 以上'},
    {value: '10', label: '10 MB 以上'},
    {value: '50', label: '50 MB 以上'},
] as const;
