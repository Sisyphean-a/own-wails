import type {HistoryWorkspaceController} from './history-workspace-controller';
import {HistoryThreadTable} from './history-workspace-thread-table';

export function SessionPanel(props: HistoryWorkspaceController) {
    if (props.loading === 'scan' && !props.listResult) {
        return (
            <section aria-label="正在加载会话" className="面板 列表面板 加载面板" role="status">
                <div className="列表表头 列表表头-对比">
                    <span/><span>当前会话</span><span>对比会话</span><span>建议</span>
                </div>
                <div className="骨架列表">
                    {Array.from({length: 6}, (_, index) => <div className="骨架行" key={index}><i/><i/><i/></div>)}
                </div>
            </section>
        );
    }
    return (
        <section className="面板 列表面板">
            <HistoryThreadTable
                allItems={props.listResult?.items ?? []}
                duplicateAnalysis={props.duplicateAnalysis}
                items={props.visibleThreads}
                selectedIds={props.selectedIds}
                toggleAllVisible={props.actions.toggleAllVisible}
                toggleSelected={props.actions.toggleSelected}
            />
        </section>
    );
}
