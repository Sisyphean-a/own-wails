import type {DeletePlanItem, DeletePlanResult, DuplicateGroup, GroupCandidate, ScanResult, WorkspaceState} from "./workspace-types";

const emptyDash = '无记录';

const actionLabels: Record<string, string> = {
    keep: '保留',
    repair_index: '修复入口',
    quarantine: '隔离',
    delete: '删除',
    archive: '归档',
};

const relationLabels: Record<string, string> = {
    preferred: '保留本',
    'path-alias': '别名',
    'physical-copy': '副本',
};

const kindLabels: Record<string, string> = {
    auth_json: '认证',
    config_toml: '配置',
    credentials_json: '凭据',
    history_jsonl: '历史',
    session_index_jsonl: '会话索引',
    state_sqlite: '状态库',
    logs_sqlite: '日志库',
    rollout_jsonl: '执行记录',
    archived_rollout_jsonl: '归档执行记录',
};

function MetricCard({label, value}: { label: string; value: string | number }) {
    return (
        <article className="指标卡">
            <span className="指标名">{label}</span>
            <strong className="指标值">{value}</strong>
        </article>
    );
}

function DataRow({label, value}: { label: string; value: string }) {
    return (
        <div className="数据行">
            <span>{label}</span>
            <code>{value || emptyDash}</code>
        </div>
    );
}

function StatusTag({tone, text}: { tone: 'neutral' | 'warn' | 'accent'; text: string }) {
    return <span className={`状态签 ${tone}`}>{text}</span>;
}

function WarningList({warnings}: { warnings: string[] }) {
    const rows = warnings ?? [];
    if (rows.length === 0) return <div className="空态 小号">无提示</div>;
    return (
        <div className="告警列">
            {rows.map((warning) => (
                <div className="告警项" key={warning}>{warning}</div>
            ))}
        </div>
    );
}

function PlanTable({items}: { items: DeletePlanItem[] }) {
    if (items.length === 0) return <div className="空态 小号">未发现重复计划项</div>;
    return (
        <div className="表格壳">
            <table className="结果表格">
                <thead>
                <tr>
                    <th>组</th>
                    <th>动作</th>
                    <th>源路径</th>
                    <th>原因</th>
                </tr>
                </thead>
                <tbody>
                {items.map((item) => (
                    <tr key={`${item.duplicateGroup}:${item.sourcePath}`}>
                        <td>{item.duplicateGroup}</td>
                        <td><StatusTag text={actionLabels[item.action] ?? item.action} tone={item.reviewNeeded ? 'warn' : 'accent'}/></td>
                        <td className="路径列"><code>{item.sourcePath}</code></td>
                        <td>{item.reason}</td>
                    </tr>
                ))}
                </tbody>
            </table>
        </div>
    );
}

function CandidateTable({candidates}: { candidates: GroupCandidate[] }) {
    return (
        <div className="表格壳">
            <table className="结果表格">
                <thead>
                <tr>
                    <th>候选</th>
                    <th>关系</th>
                    <th>动作</th>
                    <th>来源</th>
                </tr>
                </thead>
                <tbody>
                {candidates.map((candidate) => (
                    <tr key={candidate.sourcePath}>
                        <td>
                            <div className="候选主值">{candidate.preferred ? '保留' : candidate.reasonCode}</div>
                            <div className="候选副值">{candidate.reason}</div>
                        </td>
                        <td><StatusTag text={relationLabels[candidate.relation] ?? candidate.relation} tone={candidate.preferred ? 'accent' : 'neutral'}/></td>
                        <td>{actionLabels[candidate.action] ?? candidate.action}</td>
                        <td className="路径列"><code>{candidate.sourcePath}</code></td>
                    </tr>
                ))}
                </tbody>
            </table>
        </div>
    );
}

function GroupList({groups}: { groups: DuplicateGroup[] }) {
    if (groups.length === 0) return <div className="空态 小号">暂无重复组</div>;
    return (
        <div className="组列表">
            {groups.map((group) => (
                <article className="组卡" key={group.duplicateGroup}>
                    <header className="组头">
                        <div>
                            <strong>{group.duplicateGroup}</strong>
                            <div className="候选副值">{group.preferredPath}</div>
                        </div>
                        <StatusTag text={group.reviewNeeded ? '需复核' : '稳定'} tone={group.reviewNeeded ? 'warn' : 'accent'}/>
                    </header>
                    {group.warning ? <div className="告警项">{group.warning}</div> : null}
                    <CandidateTable candidates={group.candidates}/>
                </article>
            ))}
        </div>
    );
}

function ItemTable({items}: { items: ScanResult['items'] }) {
    if (items.length === 0) return <div className="空态 小号">无扫描对象</div>;
    return (
        <div className="表格壳">
            <table className="结果表格">
                <thead>
                <tr>
                    <th>类型</th>
                    <th>路径</th>
                    <th>大小</th>
                    <th>修改时间</th>
                </tr>
                </thead>
                <tbody>
                {items.slice(0, 12).map((item) => (
                    <tr key={`${item.kind}:${item.path}`}>
                        <td>{kindLabels[item.kind] ?? item.kind}</td>
                        <td className="路径列"><code>{item.path}</code></td>
                        <td>{item.size}</td>
                        <td>{item.mtimeUtc}</td>
                    </tr>
                ))}
                </tbody>
            </table>
        </div>
    );
}

function WorkspaceOverview({scan, plan}: { scan: ScanResult; plan: DeletePlanResult }) {
    return (
        <section className="双栏">
            <article className="面板">
                <header className="面板头"><h2>运行与产物</h2></header>
                <div className="数据列">
                    <DataRow label="运行 ID" value={scan.runId}/>
                    <DataRow label="发现清单" value={scan.discoveryPath}/>
                    <DataRow label="执行前清单" value={scan.manifestPath}/>
                    <DataRow label="重复组清单" value={plan.duplicateGroupsPath}/>
                    <DataRow label="删除计划" value={plan.deletePlanPath}/>
                    <DataRow label="未识别对象" value={scan.unknownItemsPath}/>
                </div>
            </article>

            <article className="面板">
                <header className="面板头">
                    <h2>计划提示</h2>
                    <StatusTag text="只读" tone="neutral"/>
                </header>
                <WarningList warnings={plan.warnings}/>
            </article>
        </section>
    );
}

function ReadyWorkspace({scan, plan}: { scan: ScanResult; plan: DeletePlanResult }) {
    return (
        <div className="结果区">
            <section className="指标网格">
                <MetricCard label="扫描根" value={scan.summary.rootCount}/>
                <MetricCard label="扫描对象" value={scan.summary.itemCount}/>
                <MetricCard label="重复组" value={plan.summary.groupCount}/>
                <MetricCard label="计划项" value={plan.summary.plannedCount}/>
                <MetricCard label="待人工复核" value={plan.summary.reviewCount}/>
                <MetricCard label="未识别对象" value={scan.summary.unknownCount}/>
            </section>
            <WorkspaceOverview plan={plan} scan={scan}/>

            <article className="面板">
                <header className="面板头">
                    <h2>删除计划</h2>
                    <StatusTag text="待确认" tone="warn"/>
                </header>
                <PlanTable items={plan.items}/>
            </article>

            <article className="面板">
                <header className="面板头"><h2>重复组复核</h2></header>
                <GroupList groups={plan.groups}/>
            </article>

            <article className="面板">
                <header className="面板头"><h2>扫描对象预览</h2></header>
                <ItemTable items={scan.items}/>
            </article>
        </div>
    );
}

export function ResultPanel({workspace}: { workspace: WorkspaceState }) {
    if (workspace.kind === 'idle') return <div className="空态">未执行</div>;
    if (workspace.kind === 'running') return <div className="空态">扫描与计划生成中</div>;
    if (workspace.kind === 'error') return <div className="错误面板">{workspace.message}</div>;
    return <ReadyWorkspace plan={workspace.plan} scan={workspace.scan}/>;
}
