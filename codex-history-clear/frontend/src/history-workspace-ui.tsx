import {X} from '@phosphor-icons/react';
import {HistoryPlanTargetTable} from './history-workspace-tables';
import type {HistoryWorkspaceController} from './history-workspace-controller';
import {DataRow, scanStatusText, ToolbarButton} from './history-workspace-toolbar';

type DialogProps = HistoryWorkspaceController & {
    open: boolean;
    onClose: () => void;
};

export function DeletePreviewDialog(props: DialogProps) {
    if (!props.open) return null;
    const plan = props.planState.planResult;
    const targetCount = plan?.summary.targetCount ?? props.selectedIds.length;
    const backdropClick = props.loading === 'execute' ? undefined : props.onClose;
    return (
        <div className="弹窗遮罩" onClick={backdropClick} role="presentation">
            <section
                aria-labelledby="删除预览标题"
                aria-modal="true"
                className="预览弹窗 面板"
                onClick={(event) => event.stopPropagation()}
                role="dialog"
            >
                <header className="弹窗头">
                    <div>
                        <h2 id="删除预览标题">删除预览</h2>
                        <p>真正清理前，先确认这次会动哪些会话和备份路径。</p>
                    </div>
                    <button className="图标控件" disabled={props.loading === 'execute'} onClick={props.onClose} title="关闭" type="button">
                        <X aria-hidden="true" size={16} weight="bold"/>
                    </button>
                </header>
                <div className="弹窗摘要">
                    <SummaryValue label="将删除" value={`${targetCount} 条会话`}/>
                    <SummaryValue label="预计释放" value={props.overview.releaseText}/>
                    <SummaryValue code label="备份位置" value={backupText(props)}/>
                </div>
                {props.planState.riskNotes.length > 0 ? (
                    <section className="预览风险">
                        <strong>执行提示</strong>
                        <ul>
                            {props.planState.riskNotes.map((item) => <li key={item}>{item}</li>)}
                        </ul>
                    </section>
                ) : null}
                <div className="预览表格">
                    {plan ? <HistoryPlanTargetTable targets={plan.targets}/> : <div className="空态 小号">还没有可展示的预览结果</div>}
                </div>
                <footer className="弹窗底栏">
                    <div className="确认说明">
                        <strong>二次确认</strong>
                        <p>输入 {props.confirmPhrase} 后，才能点击真正删除。</p>
                    </div>
                    <input
                        className="输入 确认输入"
                        onChange={(event) => props.planState.setConfirmText(event.target.value)}
                        placeholder={`输入 ${props.confirmPhrase}`}
                        value={props.planState.confirmText}
                    />
                    <div className="弹窗操作">
                        <button className="次按钮" disabled={props.loading === 'execute'} onClick={props.onClose} type="button">取消</button>
                        <button className="次按钮" disabled={!plan || props.loading === 'execute'} onClick={props.actions.backupPlan} type="button">
                            {props.loading === 'execute' ? '处理中' : '只备份不删除'}
                        </button>
                        <button className="危险按钮" disabled={!props.planState.canConfirm || props.loading === 'execute'} onClick={props.actions.executePlan} type="button">
                            {props.loading === 'execute' ? '清理中' : '确认删除'}
                        </button>
                    </div>
                </footer>
            </section>
        </div>
    );
}

export function SettingsDialog(props: DialogProps) {
    if (!props.open) return null;
    return (
        <div className="弹窗遮罩" onClick={props.onClose} role="presentation">
            <section
                aria-labelledby="设置标题"
                aria-modal="true"
                className="设置弹窗 面板"
                onClick={(event) => event.stopPropagation()}
                role="dialog"
            >
                <header className="弹窗头">
                    <div>
                        <h2 id="设置标题">设置</h2>
                        <p>目录、扫描、备份和安全策略都集中放在这里。</p>
                    </div>
                    <button className="图标控件" onClick={props.onClose} title="关闭" type="button">
                        <X aria-hidden="true" size={16} weight="bold"/>
                    </button>
                </header>
                <div className="设置内容">
                    <WorkspaceSettingsSection {...props}/>
                    <SafetySettingsSection {...props}/>
                    <ExecutionSettingsSection {...props}/>
                </div>
                <footer className="弹窗底栏 设置底栏">
                    <button className="次按钮" onClick={props.onClose} type="button">关闭</button>
                </footer>
            </section>
        </div>
    );
}

function WorkspaceSettingsSection(props: HistoryWorkspaceController) {
    return (
        <section className="设置分组">
            <div className="分组头">
                <h3>工作区</h3>
                <span className={`状态签 ${scanStatusTone(props)}`}>{scanStatusText(props)}</span>
            </div>
            <div className="数据列">
                <DataRow code label="当前目录" value={props.workspaceConfig?.codexHome ?? '读取中'}/>
                <DataRow code label="备份根目录" value={props.workspaceConfig?.backupRoot ?? '未配置备份目录'}/>
                <DataRow code label="本次备份" value={backupText(props)}/>
            </div>
            <div className="设置操作组">
                <ToolbarButton disabled={props.loading === 'directory'} icon="folder" onClick={props.actions.changeDirectory} text="更换目录"/>
                <ToolbarButton icon="archive" onClick={props.actions.openBackupDirectory} text="打开备份目录"/>
                <ToolbarButton disabled={props.loading === 'scan'} icon="refresh" onClick={props.actions.startScan} text={scanActionText(props)} tone="primary"/>
            </div>
        </section>
    );
}

function SafetySettingsSection(props: HistoryWorkspaceController) {
    return (
        <section className="设置分组">
            <div className="分组头">
                <h3>安全选项</h3>
                <span className="状态签 neutral">{props.planState.affectedProjects.length} 个项目</span>
            </div>
            <div className="设置勾选网格">
                <CheckboxField checked={props.strategyState.autoBackup} label="删除前自动备份" onChange={props.strategyState.setAutoBackup}/>
                <CheckboxField checked={props.strategyState.generateReport} label="生成校验报告" onChange={props.strategyState.setGenerateReport}/>
                <CheckboxField checked={props.strategyState.keepRecent} label="保留最近 30 天" onChange={props.strategyState.setKeepRecent}/>
                <CheckboxField checked={props.strategyState.skipUnknown} label="跳过未知结构" onChange={props.strategyState.setSkipUnknown}/>
            </div>
        </section>
    );
}

function ExecutionSettingsSection(props: HistoryWorkspaceController) {
    const execution = props.planState.executionResult;
    if (!execution) return null;
    return (
        <section className="设置分组">
            <div className="分组头">
                <h3>执行后操作</h3>
                <span className={`状态签 ${execution.verification.success ? 'accent' : 'warn'}`}>{execution.verification.summary}</span>
            </div>
            <div className="数据列">
                <DataRow code label="恢复记录" value={execution.rollbackJournalPath || '未保留恢复点'}/>
                <DataRow code label="执行结果" value={execution.execResultPath}/>
            </div>
            <div className="设置操作组">
                <button
                    className="次按钮"
                    disabled={props.loading === 'rollback' || execution.rollbackJournalPath === ''}
                    onClick={props.actions.rollbackPlan}
                    type="button"
                >
                    按备份恢复
                </button>
                <button className="次按钮" disabled={props.loading === 'export'} onClick={props.actions.exportEvidencePack} type="button">
                    导出报告
                </button>
            </div>
        </section>
    );
}

function CheckboxField({checked, label, onChange}: { checked: boolean; label: string; onChange: (value: boolean) => void }) {
    return <label className="勾选项"><input checked={checked} onChange={(event) => onChange(event.target.checked)} type="checkbox"/><span>{label}</span></label>;
}

export function ScanWarningsDialog(props: DialogProps) {
	if (!props.open) return null;
	const warnings = props.listResult?.warnings ?? [];
	return (
		<div className="弹窗遮罩" onClick={props.onClose} role="presentation">
			<section aria-labelledby="扫描告警标题" aria-modal="true" className="设置弹窗 面板" onClick={(event) => event.stopPropagation()} role="dialog">
				<header className="弹窗头">
					<div><h2 id="扫描告警标题">扫描告警</h2><p>这些转录没有被静默忽略，可按路径检查原始文件。</p></div>
					<button className="图标控件" onClick={props.onClose} title="关闭" type="button"><X aria-hidden="true" size={16} weight="bold"/></button>
				</header>
				<div className="告警列表">
					{warnings.map((warning) => (
						<div className="告警项" key={`${warning.path}:${warning.code}`}>
							<div><strong>{warning.code}</strong><span>{warning.message}</span></div>
							<code title={warning.path}>{warning.path}</code>
						</div>
					))}
				</div>
				<footer className="弹窗底栏 设置底栏"><button className="次按钮" onClick={props.onClose} type="button">关闭</button></footer>
			</section>
		</div>
	);
}

function SummaryValue({label, value, code = false}: { label: string; value: string; code?: boolean }) {
    return (
        <div className={`摘要槽 ${code ? '摘要槽-路径' : ''}`}>
            <span>{label}</span>
            {code ? <code title={value}>{value}</code> : <strong title={value}>{value}</strong>}
        </div>
    );
}


function scanActionText(props: HistoryWorkspaceController) {
    if (props.loading === 'scan') return '扫描中';
    if (props.scanWorkspace.kind === 'idle') return '开始扫描';
    return '重新扫描';
}

function scanStatusTone(props: HistoryWorkspaceController): 'accent' | 'warn' | 'neutral' {
    if (props.loading === 'scan' || props.scanWorkspace.kind === 'ready') return 'accent';
    if (props.scanWorkspace.kind === 'error') return 'warn';
    return 'neutral';
}

function backupText(props: HistoryWorkspaceController) {
    return props.planState.backupPath || props.workspaceConfig?.backupRoot || '未配置备份目录';
}
