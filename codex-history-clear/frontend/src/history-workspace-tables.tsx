import type {HistoryExecutionResult, HistoryPlanResult} from './history-types';

const actionLabels: Record<string, string> = {
    delete_rows: '删行',
    rewrite_rows: '清引用',
    rewrite_jsonl: '改 JSONL',
    rewrite_json: '改 JSON',
    delete_file: '删文件',
    inspect: '检查',
};

export function HistoryPlanTargetTable({targets}: { targets: HistoryPlanResult['targets'] }) {
    if (targets.length === 0) return <div className="空态 小号">当前范围没有可执行目标</div>;
    return (
        <div className="表格壳">
            <table className="结果表格">
                <thead>
                <tr>
                    <th>会话</th>
                    <th>路径</th>
                    <th>动作数</th>
                    <th>提示</th>
                </tr>
                </thead>
                <tbody>
                {targets.map((target) => (
                    <tr key={target.thread.id}>
                        <td>
                            <div className="候选主值" title={target.thread.title || '未命名会话'}>{target.thread.title || '未命名会话'}</div>
                            <div className="候选副值">{shortID(target.thread.id)}</div>
                        </td>
                        <td className="路径列"><code title={target.thread.rolloutPath || '无路径'}>{target.thread.rolloutPath || '无路径'}</code></td>
                        <td>{target.stores.filter((store) => store.count > 0 || store.store === 'rollout_jsonl').length}</td>
                        <td title={target.warnings[0] ?? '无'}>{target.warnings[0] ?? '无'}</td>
                    </tr>
                ))}
                </tbody>
            </table>
        </div>
    );
}

export function HistoryExecutionTable({result}: { result: HistoryExecutionResult }) {
    return (
        <div className="结果区">
            <ExecutionSummary result={result}/>
            <VerificationTable result={result}/>
            <MutationTable result={result}/>
            <EventTable result={result}/>
        </div>
    );
}

function ExecutionSummary({result}: { result: HistoryExecutionResult }) {
    return (
        <>
            <section className="指标网格 三列">
                <article className="指标卡"><span className="指标名">改动条目</span><strong className="指标值">{result.mutations.filter((item) => item.changed).length}</strong></article>
                <article className="指标卡"><span className="指标名">备份文件</span><strong className="指标值">{result.backups.length}</strong></article>
                <article className="指标卡"><span className="指标名">残留引用</span><strong className="指标值">{result.verification.remainingReferences.length}</strong></article>
            </section>
            <div className="数据列">
                <div className="数据行"><span>执行模式</span><code title={result.mode}>{result.mode}</code></div>
                <div className="数据行"><span>校验结论</span><code title={result.verification.summary}>{result.verification.summary}</code></div>
                <div className="数据行"><span>已确认计划</span><code title={result.approvedPlanPath}>{result.approvedPlanPath}</code></div>
                <div className="数据行"><span>恢复记录</span><code title={result.rollbackJournalPath}>{result.rollbackJournalPath}</code></div>
                <div className="数据行"><span>执行结果</span><code title={result.execResultPath}>{result.execResultPath}</code></div>
                <div className="数据行"><span>清理后清单</span><code title={result.manifestAfterPath}>{result.manifestAfterPath}</code></div>
            </div>
        </>
    );
}

function VerificationTable({result}: { result: HistoryExecutionResult }) {
    if (result.verification.remainingReferences.length === 0) return null;
    return (
        <div className="表格壳">
            <table className="结果表格">
                <thead>
                <tr>
                    <th>残留存储</th>
                    <th>说明</th>
                    <th>路径</th>
                </tr>
                </thead>
                <tbody>
                {result.verification.remainingReferences.map((item) => (
                    <tr key={`${item.store}:${item.path}`}>
                        <td>{item.store}</td>
                        <td title={item.detail}>{item.detail}</td>
                        <td className="路径列"><code title={item.path}>{item.path}</code></td>
                    </tr>
                ))}
                </tbody>
            </table>
        </div>
    );
}

function MutationTable({result}: { result: HistoryExecutionResult }) {
    return (
        <div className="表格壳">
            <table className="结果表格">
                <thead>
                <tr>
                    <th>存储</th>
                    <th>动作</th>
                    <th>变化</th>
                    <th>路径</th>
                </tr>
                </thead>
                <tbody>
                {result.mutations.map((item) => (
                    <tr key={`${item.store}:${item.path}`}>
                        <td>{item.store}</td>
                        <td>{actionLabels[item.action] ?? item.action}</td>
                        <td>{item.changedRows}</td>
                        <td className="路径列"><code title={item.path}>{item.path}</code></td>
                    </tr>
                ))}
                </tbody>
            </table>
        </div>
    );
}

function EventTable({result}: { result: HistoryExecutionResult }) {
    return (
        <div className="表格壳">
            <table className="结果表格">
                <thead>
                <tr>
                    <th>阶段</th>
                    <th>级别</th>
                    <th>消息</th>
                    <th>产物</th>
                </tr>
                </thead>
                <tbody>
                {result.events.map((item, index) => (
                    <tr key={`${item.phase}:${index}`}>
                        <td>{item.phase}</td>
                        <td>{item.level}</td>
                        <td title={item.message}>{item.message}</td>
                        <td className="路径列"><code title={item.artifactPath}>{item.artifactPath}</code></td>
                    </tr>
                ))}
                </tbody>
            </table>
        </div>
    );
}

function shortID(value: string) {
    return value.slice(0, 8);
}
