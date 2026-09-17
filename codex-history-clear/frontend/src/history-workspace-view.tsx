import {useEffect, useState} from 'react';
import type {HistoryWorkspaceController} from './history-workspace-controller';
import {SessionPanel} from './history-workspace-panels';
import {ToolbarPanel} from './history-workspace-toolbar';
import {DeletePreviewDialog, ScanWarningsDialog, SettingsDialog} from './history-workspace-ui';

export function HistoryWorkspaceView(props: HistoryWorkspaceController) {
    const [previewOpen, setPreviewOpen] = useState(false);
    const [previewPending, setPreviewPending] = useState(false);
	const [settingsOpen, setSettingsOpen] = useState(false);
	const [warningsOpen, setWarningsOpen] = useState(false);

    useEffect(() => {
        if (!previewPending || props.loading === 'plan') return;
        setPreviewPending(false);
        if (props.planState.planResult) setPreviewOpen(true);
    }, [previewPending, props.loading, props.planState.planResult]);

    useEffect(() => {
        if (!previewOpen || !props.planState.executionResult) return;
        props.planState.setConfirmText('');
        setPreviewOpen(false);
    }, [previewOpen, props.planState.executionResult]);

    const closePreview = () => {
        props.planState.setConfirmText('');
        setPreviewOpen(false);
    };

    const openPreview = async () => {
        if (props.selectedIds.length === 0 || props.loading === 'plan') return;
        setSettingsOpen(false);
        if (props.planState.planResult) {
            props.planState.setConfirmText('');
            setPreviewOpen(true);
            return;
        }
        setPreviewPending(true);
        await props.actions.buildPlan();
    };

    return (
        <article className="工具壳">
            <section className="顶部区">
                <ToolbarPanel
					{...props}
					onOpenPreview={openPreview}
					onOpenSettings={() => setSettingsOpen(true)}
					onOpenWarnings={() => setWarningsOpen(true)}
                />
                {props.error ? <div className="错误横幅">{props.error}</div> : null}
            </section>
            <section className="主工作区">
                <section className="主内容列">
                    <SessionPanel {...props}/>
                </section>
            </section>
            <DeletePreviewDialog {...props} onClose={closePreview} open={previewOpen}/>
			<SettingsDialog {...props} onClose={() => setSettingsOpen(false)} open={settingsOpen}/>
			<ScanWarningsDialog {...props} onClose={() => setWarningsOpen(false)} open={warningsOpen}/>
        </article>
    );
}
