import './history-workspace.css';
import './history-workspace-controls.css';
import './history-workspace-dialogs.css';
import './history-workspace-list.css';
import {useHistoryWorkspaceController} from './history-workspace-controller';
import {HistoryWorkspaceView} from './history-workspace-view';

export function HistoryWorkspace() {
    const controller = useHistoryWorkspaceController();
    return <HistoryWorkspaceView {...controller}/>;
}
